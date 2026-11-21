package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/Gpay-Remit/gpay-remit/config"
)

func TestExchangeRateHandler_ReportsRateSourceHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oracle := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","rates":{"EUR":0.9}}`))
	}))
	defer oracle.Close()

	// Redis is not initialised in unit tests, so the service always reaches the
	// oracle; the header must say so.
	h := NewExchangeRateHandler(&config.Config{ExchangeRateAPIURL: oracle.URL})

	router := gin.New()
	router.GET("/exchange-rates", h.GetRate)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/exchange-rates?from=usd&to=eur", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "oracle", w.Header().Get("X-Exchange-Rate-Source"))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "USD", body["from"])
	assert.Equal(t, "EUR", body["to"])
	assert.Equal(t, float64(900000), body["rate"])
	assert.NotContains(t, body, "Origin", "internal origin field must not leak into the body")
}
