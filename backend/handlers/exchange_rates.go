package handlers

import (
	"net/http"
	"strings"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/Gpay-Remit/gpay-remit/errors"
	"github.com/Gpay-Remit/gpay-remit/services"
	"github.com/gin-gonic/gin"
)

type ExchangeRateHandler struct {
	service *services.ExchangeRateService
}

func NewExchangeRateHandler(cfg *config.Config) *ExchangeRateHandler {
	return &ExchangeRateHandler{
		service: services.NewExchangeRateService(cfg),
	}
}

// GetRate handles GET /exchange-rates?from=USD&to=EUR
//
// Caching, oracle fallback and the circuit breaker live in the service. The
// X-Exchange-Rate-Source response header reports whether the rate came from
// cache, the oracle, or a stale copy served during an outage.
func (h *ExchangeRateHandler) GetRate(c *gin.Context) {
	from := strings.ToUpper(strings.TrimSpace(c.Query("from")))
	to := strings.ToUpper(strings.TrimSpace(c.Query("to")))

	if from == "" || to == "" {
		c.Error(errors.NewValidationError("from and to are required", "missing from/to query params"))
		return
	}

	if len(from) != 3 || len(to) != 3 {
		c.Error(errors.NewValidationError("invalid currency code", "from/to must be 3-letter ISO currency codes"))
		return
	}

	rate, err := h.service.GetRate(from, to)
	if err != nil {
		c.Error(errors.NewInternalError("Failed to retrieve exchange rate", err))
		return
	}

	c.Header("X-Exchange-Rate-Source", rate.Origin)
	c.JSON(http.StatusOK, rate)
}
