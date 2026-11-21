package handlers

import (
	"math"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/Gpay-Remit/gpay-remit/errors"
	"github.com/Gpay-Remit/gpay-remit/services"
)

type FeeHandler struct {
	fees *services.FeeService
}

func NewFeeHandler(fees *services.FeeService) *FeeHandler {
	return &FeeHandler{fees: fees}
}

func (h *FeeHandler) Calculate(c *gin.Context) {
	amountStr := c.Query("amount")
	if amountStr == "" {
		c.Error(errors.NewValidationError("amount is required", "missing amount query param"))
		return
	}

	var amount int64
	if _, err := fmt.Sscanf(amountStr, "%d", &amount); err != nil || amount <= 0 {
		var floatVal float64
		if _, errF := fmt.Sscanf(amountStr, "%f", &floatVal); errF == nil && floatVal > 0 {
			amount = int64(math.Round(floatVal * 100))
		} else {
			c.Error(errors.NewValidationError("invalid amount", "amount must be a positive integer in minor units (cents/stroops)"))
			return
		}
	}

	breakdown := h.fees.Calculate(amount)
	c.JSON(http.StatusOK, breakdown)
}
