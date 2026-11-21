package services

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRateCache mimics the Redis cache: values are JSON round-tripped, and
// tests can delete a key to simulate its TTL expiring.
type fakeRateCache struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newFakeRateCache() *fakeRateCache {
	return &fakeRateCache{data: make(map[string][]byte)}
}

func (c *fakeRateCache) Get(key string, dest interface{}) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, ok := c.data[key]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(raw, dest)
}

func (c *fakeRateCache) Set(key string, value interface{}, _ time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = raw
	return nil
}

// expireFresh removes the fresh (TTL-bound) entry for a pair but leaves the
// stale copy, which is what Redis would look like after the fresh TTL passes.
func (c *fakeRateCache) expireFresh(from, to string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, rateCacheKey(from, to))
}

// missCache never stores anything, simulating Redis being unavailable.
type missCache struct{}

func (missCache) Get(string, interface{}) (bool, error)        { return false, nil }
func (missCache) Set(string, interface{}, time.Duration) error { return nil }

// oracleStub is a switchable upstream: healthy serves rates, down returns 500,
// and calls counts every request that reaches it.
type oracleStub struct {
	healthy atomic.Bool
	calls   atomic.Int64
	rates   map[string]float64
	server  *httptest.Server
}

func newOracleStub(t *testing.T, rates map[string]float64) *oracleStub {
	t.Helper()
	o := &oracleStub{rates: rates}
	o.healthy.Store(true)
	o.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.calls.Add(1)
		if !o.healthy.Load() {
			http.Error(w, "upstream down", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(oracleResponse{Result: "success", Rates: o.rates})
	}))
	t.Cleanup(o.server.Close)
	return o
}

// fakeClock lets tests move time forward past the breaker cooldown and the
// stale retention window without sleeping.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestExchangeRateService(t *testing.T, oracle *oracleStub, cache interface {
	Get(string, interface{}) (bool, error)
	Set(string, interface{}, time.Duration) error
}) (*ExchangeRateService, *fakeClock) {
	t.Helper()
	svc := newExchangeRateService(oracle.server.URL, cache, 60*time.Second, &http.Client{Timeout: 2 * time.Second})
	clock := &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	svc.now = clock.Now
	return svc, clock
}

func TestGetRate_CacheHitAvoidsOracle(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	svc, _ := newTestExchangeRateService(t, oracle, newFakeRateCache())

	first, err := svc.GetRate("usd", "eur")
	require.NoError(t, err)
	assert.Equal(t, RateSourceOracle, first.Origin)
	assert.Equal(t, int64(900000), first.Rate)
	assert.Equal(t, "USD", first.From)
	assert.Equal(t, "EUR", first.To)

	second, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)
	assert.Equal(t, RateSourceCache, second.Origin)
	assert.Equal(t, first.Rate, second.Rate)

	assert.Equal(t, int64(1), oracle.calls.Load(), "fresh cache hit must not reach the oracle")
}

func TestGetRate_ExpiredFreshEntryRefetchesFromOracle(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	cache := newFakeRateCache()
	svc, _ := newTestExchangeRateService(t, oracle, cache)

	_, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)

	cache.expireFresh("USD", "EUR")

	rate, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)
	assert.Equal(t, RateSourceOracle, rate.Origin)
	assert.Equal(t, int64(2), oracle.calls.Load())
}

func TestGetRate_ConfiguredTTLIsUsed(t *testing.T) {
	cfg := &config.Config{ExchangeRateAPIURL: "http://example.invalid", ExchangeRateCacheTTL: 0}
	assert.Equal(t, DefaultExchangeRateCacheTTL, NewExchangeRateService(cfg).cacheTTL, "zero TTL falls back to the 60s default")

	cfg.ExchangeRateCacheTTL = 15 * time.Second
	assert.Equal(t, 15*time.Second, NewExchangeRateService(cfg).cacheTTL)
}

func TestGetRate_OracleDownServesStaleRate(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	cache := newFakeRateCache()
	svc, clock := newTestExchangeRateService(t, oracle, cache)

	good, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)

	oracle.healthy.Store(false)
	cache.expireFresh("USD", "EUR")
	clock.Advance(5 * time.Minute)

	stale, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err, "oracle outage must not fail the lookup when a stale rate exists")
	assert.Equal(t, RateSourceStale, stale.Origin)
	assert.Equal(t, good.Rate, stale.Rate)
	assert.Equal(t, good.Timestamp.Unix(), stale.Timestamp.Unix(), "stale rate keeps its original timestamp")
}

func TestGetRate_OracleDownWithNoPriorRateErrors(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	oracle.healthy.Store(false)
	svc, _ := newTestExchangeRateService(t, oracle, newFakeRateCache())

	rate, err := svc.GetRate("USD", "EUR")
	assert.Error(t, err)
	assert.Nil(t, rate)
}

func TestGetRate_StaleRateOlderThanRetentionIsRefused(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	cache := newFakeRateCache()
	svc, clock := newTestExchangeRateService(t, oracle, cache)

	_, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)

	oracle.healthy.Store(false)
	cache.expireFresh("USD", "EUR")
	clock.Advance(staleRateRetention + time.Minute)

	rate, err := svc.GetRate("USD", "EUR")
	assert.Error(t, err)
	assert.Nil(t, rate)
}

func TestGetRate_InProcessFallbackWhenRedisUnavailable(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"GBP": 0.75})
	svc, _ := newTestExchangeRateService(t, oracle, missCache{})

	// Without Redis every call is a miss, so the first call populates the
	// in-process last-known-good map and later calls go to the oracle.
	_, err := svc.GetRate("USD", "GBP")
	require.NoError(t, err)

	oracle.healthy.Store(false)
	stale, err := svc.GetRate("USD", "GBP")
	require.NoError(t, err)
	assert.Equal(t, RateSourceStale, stale.Origin)
	assert.Equal(t, int64(750000), stale.Rate)
}

func TestGetRate_CircuitBreakerStopsCallingFailingOracle(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	cache := newFakeRateCache()
	svc, clock := newTestExchangeRateService(t, oracle, cache)

	// Prime a good rate so outages can be served stale.
	_, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)
	require.Equal(t, int64(1), oracle.calls.Load())

	oracle.healthy.Store(false)

	// Each of these misses the fresh cache and hits the failing oracle until
	// the breaker threshold is reached.
	for i := 0; i < breakerFailureThreshold; i++ {
		cache.expireFresh("USD", "EUR")
		rate, err := svc.GetRate("USD", "EUR")
		require.NoError(t, err)
		assert.Equal(t, RateSourceStale, rate.Origin)
	}
	callsAtOpen := oracle.calls.Load()
	assert.Equal(t, int64(1+breakerFailureThreshold), callsAtOpen)

	// Breaker is open: further misses are served stale without touching the oracle.
	for i := 0; i < 5; i++ {
		cache.expireFresh("USD", "EUR")
		rate, err := svc.GetRate("USD", "EUR")
		require.NoError(t, err)
		assert.Equal(t, RateSourceStale, rate.Origin)
	}
	assert.Equal(t, callsAtOpen, oracle.calls.Load(), "open breaker must not call the oracle")

	// After the cooldown exactly one trial call goes out; it fails, so the
	// breaker reopens for another full cooldown.
	clock.Advance(breakerCooldown + time.Second)
	cache.expireFresh("USD", "EUR")
	_, err = svc.GetRate("USD", "EUR")
	require.NoError(t, err)
	assert.Equal(t, callsAtOpen+1, oracle.calls.Load(), "half-open allows exactly one trial call")

	cache.expireFresh("USD", "EUR")
	_, err = svc.GetRate("USD", "EUR")
	require.NoError(t, err)
	assert.Equal(t, callsAtOpen+1, oracle.calls.Load(), "failed trial reopens the breaker")
}

func TestGetRate_CircuitBreakerRecoversAfterCooldown(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	cache := newFakeRateCache()
	svc, clock := newTestExchangeRateService(t, oracle, cache)

	_, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)

	oracle.healthy.Store(false)
	for i := 0; i < breakerFailureThreshold; i++ {
		cache.expireFresh("USD", "EUR")
		_, err := svc.GetRate("USD", "EUR")
		require.NoError(t, err)
	}

	oracle.healthy.Store(true)
	clock.Advance(breakerCooldown + time.Second)
	cache.expireFresh("USD", "EUR")

	rate, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)
	assert.Equal(t, RateSourceOracle, rate.Origin, "successful trial returns to the oracle")
	assert.Equal(t, 0, svc.breaker.consecutiveFailures)
	assert.True(t, svc.breaker.openUntil.IsZero())
}

func TestGetRate_MissingPairDoesNotTripBreakerOrServeStale(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	cache := newFakeRateCache()
	svc, _ := newTestExchangeRateService(t, oracle, cache)

	// Prime a rate for USD->EUR so stale data exists for this currency.
	_, err := svc.GetRate("USD", "EUR")
	require.NoError(t, err)

	for i := 0; i < breakerFailureThreshold+2; i++ {
		rate, err := svc.GetRate("USD", "XYZ")
		assert.True(t, errors.Is(err, ErrRateNotFound), "got %v", err)
		assert.Nil(t, rate, "a missing pair must not be answered with another pair's rate")
	}
	assert.Equal(t, 0, svc.breaker.consecutiveFailures)
}

func TestGetRate_InvalidOracleRateRejected(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0})
	svc, _ := newTestExchangeRateService(t, oracle, newFakeRateCache())

	rate, err := svc.GetRate("USD", "EUR")
	assert.Error(t, err)
	assert.Nil(t, rate)
}

func TestGetRate_ConcurrentMissesShareOneOracleCall(t *testing.T) {
	oracle := newOracleStub(t, map[string]float64{"EUR": 0.9})
	svc, _ := newTestExchangeRateService(t, oracle, newFakeRateCache())

	const workers = 50
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.GetRate("USD", "EUR"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected error: %v", err)
	}
	assert.Equal(t, int64(1), oracle.calls.Load(), "cold-cache stampede must produce a single oracle call")
}

func TestCircuitBreaker_States(t *testing.T) {
	var b circuitBreaker
	now := time.Now()

	assert.True(t, b.allow(now), "closed breaker allows calls")
	for i := 0; i < breakerFailureThreshold-1; i++ {
		b.recordFailure(now)
	}
	assert.True(t, b.allow(now), "below threshold still closed")

	b.recordFailure(now)
	assert.False(t, b.allow(now), "threshold reached opens breaker")
	assert.False(t, b.allow(now.Add(breakerCooldown-time.Millisecond)))
	assert.True(t, b.allow(now.Add(breakerCooldown)), "cooldown elapsed allows a trial")

	b.recordSuccess()
	assert.True(t, b.allow(now))
	assert.Equal(t, 0, b.consecutiveFailures)
}
