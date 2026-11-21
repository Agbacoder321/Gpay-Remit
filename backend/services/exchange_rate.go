package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/Gpay-Remit/gpay-remit/logger"
	"github.com/Gpay-Remit/gpay-remit/metrics"
	"github.com/Gpay-Remit/gpay-remit/utils"
)

const RateScale = 1000000 // 1e6 scale for integer-scaled rates

const (
	// DefaultExchangeRateCacheTTL applies when EXCHANGE_RATE_CACHE_TTL is unset.
	DefaultExchangeRateCacheTTL = 60 * time.Second

	// staleRateRetention bounds how old a rate may be and still be served
	// while the oracle is unavailable.
	staleRateRetention = 24 * time.Hour

	// Circuit breaker: after this many consecutive oracle failures, stop
	// calling the oracle for breakerCooldown and serve stale rates instead.
	breakerFailureThreshold = 3
	breakerCooldown         = 30 * time.Second
)

// Where a returned rate came from. Logged, counted in metrics, and exposed to
// clients via the X-Exchange-Rate-Source header.
const (
	RateSourceCache  = "cache"
	RateSourceOracle = "oracle"
	RateSourceStale  = "stale"
)

// ErrRateNotFound means the oracle answered but has no rate for the requested
// pair. It is a client-side problem, so it does not trip the circuit breaker
// and never falls back to a stale rate.
var ErrRateNotFound = errors.New("no rate available")

// ExchangeRate is the normalized response returned to API consumers.
type ExchangeRate struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Rate      int64     `json:"rate"`       // Integer-scaled rate (rate * 1,000,000)
	RateFloat float64   `json:"rate_float"` // Decimal rate for reference
	Scale     int64     `json:"scale"`      // Scale multiplier (1,000,000)
	Timestamp time.Time `json:"timestamp"`  // When the oracle produced this rate
	Source    string    `json:"source"`     // Oracle base URL

	// Origin records how this particular lookup was served (cache, oracle or
	// stale). It describes the response, not the stored rate, so it is not
	// serialised into the cache.
	Origin string `json:"-"`
}

type oracleResponse struct {
	Result string             `json:"result"`
	Rates  map[string]float64 `json:"rates"`
}

// ExchangeRateService returns currency conversion rates. Lookups are served
// from a short-lived cache when possible; the oracle is consulted only on a
// miss. When the oracle is failing, a circuit breaker stops the calls and the
// last known good rate is served instead.
type ExchangeRateService struct {
	baseURL    string
	httpClient *http.Client
	cache      utils.Cache
	cacheTTL   time.Duration
	now        func() time.Time

	// fetchMu serialises oracle calls and guards breaker. Holding it across
	// the miss path means concurrent misses for a pair produce one upstream
	// request instead of a stampede.
	fetchMu sync.Mutex
	breaker circuitBreaker

	// lastGood is an in-process copy of the most recent successful rate per
	// pair. It backs stale serving when Redis is unavailable.
	lastGoodMu sync.RWMutex
	lastGood   map[string]ExchangeRate
}

func NewExchangeRateService(cfg *config.Config) *ExchangeRateService {
	ttl := cfg.ExchangeRateCacheTTL
	if ttl <= 0 {
		ttl = DefaultExchangeRateCacheTTL
	}
	return newExchangeRateService(
		cfg.ExchangeRateAPIURL,
		utils.RedisCache{},
		ttl,
		&http.Client{Timeout: 10 * time.Second},
	)
}

func newExchangeRateService(baseURL string, cache utils.Cache, ttl time.Duration, client *http.Client) *ExchangeRateService {
	return &ExchangeRateService{
		baseURL:    baseURL,
		httpClient: client,
		cache:      cache,
		cacheTTL:   ttl,
		now:        time.Now,
		lastGood:   make(map[string]ExchangeRate),
	}
}

// GetRate returns the conversion rate from one currency to another.
//
// Order of resolution:
//  1. fresh cache entry (Redis, TTL = cacheTTL)
//  2. oracle, if the circuit breaker allows it
//  3. stale rate (Redis or in-process, at most staleRateRetention old), if the
//     oracle call failed or the breaker is open
func (s *ExchangeRateService) GetRate(from, to string) (*ExchangeRate, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	key := rateCacheKey(from, to)

	if rate, ok := s.readFresh(key); ok {
		return s.served(rate, RateSourceCache, from, to), nil
	}

	s.fetchMu.Lock()
	defer s.fetchMu.Unlock()

	// Another request may have filled the cache while we waited for the lock.
	if rate, ok := s.readFresh(key); ok {
		return s.served(rate, RateSourceCache, from, to), nil
	}

	if !s.breaker.allow(s.now()) {
		return s.serveStale(from, to, errors.New("exchange rate oracle circuit open"))
	}

	rate, err := s.fetchFromOracle(from, to)
	if err != nil {
		if errors.Is(err, ErrRateNotFound) {
			logger.Log.WithField("from", from).WithField("to", to).WithError(err).Warn("Exchange rate pair not available from oracle")
			return nil, err
		}
		s.breaker.recordFailure(s.now())
		logger.Log.WithField("from", from).WithField("to", to).WithError(err).
			WithField("consecutive_failures", s.breaker.consecutiveFailures).
			Warn("Exchange rate oracle call failed")
		return s.serveStale(from, to, err)
	}

	s.breaker.recordSuccess()
	s.store(key, from, to, rate)
	return s.served(rate, RateSourceOracle, from, to), nil
}

// fetchFromOracle performs one HTTP call to the oracle. Failures that mean
// "the oracle is unhealthy" are returned as plain errors; a missing pair is
// wrapped in ErrRateNotFound.
func (s *ExchangeRateService) fetchFromOracle(from, to string) (ExchangeRate, error) {
	url := fmt.Sprintf("%s/%s", strings.TrimRight(s.baseURL, "/"), from)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return ExchangeRate{}, fmt.Errorf("failed to reach exchange rate provider: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ExchangeRate{}, fmt.Errorf("exchange rate provider returned status %d", resp.StatusCode)
	}

	var parsed oracleResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ExchangeRate{}, fmt.Errorf("failed to decode exchange rate response: %w", err)
	}

	rate, ok := parsed.Rates[to]
	if !ok {
		return ExchangeRate{}, fmt.Errorf("%w: %s -> %s", ErrRateNotFound, from, to)
	}
	if !(rate > 0) || math.IsInf(rate, 0) {
		return ExchangeRate{}, fmt.Errorf("exchange rate provider returned invalid rate %v for %s -> %s", rate, from, to)
	}

	return ExchangeRate{
		From:      from,
		To:        to,
		Rate:      int64(math.Round(rate * float64(RateScale))),
		RateFloat: rate,
		Scale:     RateScale,
		Timestamp: s.now(),
		Source:    s.baseURL,
	}, nil
}

// serveStale returns the most recent usable stale rate for the pair, or an
// error if none exists or the newest one is past staleRateRetention.
func (s *ExchangeRateService) serveStale(from, to string, cause error) (*ExchangeRate, error) {
	key := rateCacheKey(from, to)
	now := s.now()

	var best *ExchangeRate
	consider := func(r ExchangeRate) {
		if best == nil || r.Timestamp.After(best.Timestamp) {
			c := r
			best = &c
		}
	}

	var fromRedis ExchangeRate
	if found, err := s.cache.Get(staleCacheKey(from, to), &fromRedis); err == nil && found {
		consider(fromRedis)
	}

	s.lastGoodMu.RLock()
	if r, ok := s.lastGood[key]; ok {
		consider(r)
	}
	s.lastGoodMu.RUnlock()

	if best == nil || now.Sub(best.Timestamp) > staleRateRetention {
		logger.Log.WithField("from", from).WithField("to", to).WithError(cause).
			Error("Exchange rate unavailable: oracle down and no usable stale rate")
		return nil, fmt.Errorf("exchange rate unavailable: %w", cause)
	}

	logger.Log.WithField("from", from).WithField("to", to).
		WithField("rate_age", now.Sub(best.Timestamp).String()).
		WithError(cause).
		Warn("Serving stale exchange rate while oracle is unavailable")
	return s.served(*best, RateSourceStale, from, to), nil
}

// readFresh returns the cached rate for key, if one exists.
func (s *ExchangeRateService) readFresh(key string) (ExchangeRate, bool) {
	var rate ExchangeRate
	found, err := s.cache.Get(key, &rate)
	if err != nil || !found {
		return ExchangeRate{}, false
	}
	return rate, true
}

// store writes a freshly fetched rate to the fresh cache, the stale cache and
// the in-process last-known-good map.
func (s *ExchangeRateService) store(key, from, to string, rate ExchangeRate) {
	_ = s.cache.Set(key, rate, s.cacheTTL)
	_ = s.cache.Set(staleCacheKey(from, to), rate, staleRateRetention)

	s.lastGoodMu.Lock()
	s.lastGood[key] = rate
	s.lastGoodMu.Unlock()
}

// served tags the rate with how it was obtained, logs the source, and counts
// it. Cache hits are logged at debug to keep steady-state traffic quiet.
func (s *ExchangeRateService) served(rate ExchangeRate, source, from, to string) *ExchangeRate {
	rate.Origin = source
	metrics.ExchangeRateLookupsTotal.WithLabelValues(source).Inc()

	entry := logger.Log.WithField("from", from).WithField("to", to).
		WithField("rate_source", source).
		WithField("rate", rate.RateFloat)
	switch source {
	case RateSourceCache:
		entry.Debug("Exchange rate served from cache")
	case RateSourceStale:
		// Already logged as a warning in serveStale with the age.
	default:
		entry.Info("Exchange rate fetched from oracle")
	}

	return &rate
}

func rateCacheKey(from, to string) string {
	return fmt.Sprintf("exchange-rate:%s:%s", from, to)
}

func staleCacheKey(from, to string) string {
	return fmt.Sprintf("exchange-rate-stale:%s:%s", from, to)
}

// circuitBreaker is a minimal three-state breaker. It is not safe for
// concurrent use; ExchangeRateService guards it with fetchMu.
//
//	closed    consecutiveFailures < threshold: calls allowed
//	open      failures >= threshold and now < openUntil: calls refused
//	half-open openUntil passed: one trial call is allowed; success closes the
//	          breaker, failure reopens it for another cooldown
type circuitBreaker struct {
	consecutiveFailures int
	openUntil           time.Time
}

func (b *circuitBreaker) allow(now time.Time) bool {
	return b.openUntil.IsZero() || !now.Before(b.openUntil)
}

func (b *circuitBreaker) recordSuccess() {
	b.consecutiveFailures = 0
	b.openUntil = time.Time{}
}

func (b *circuitBreaker) recordFailure(now time.Time) {
	b.consecutiveFailures++
	if b.consecutiveFailures >= breakerFailureThreshold {
		b.openUntil = now.Add(breakerCooldown)
	}
}
