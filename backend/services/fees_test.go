package services

import (
	"testing"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/stretchr/testify/assert"
)

func TestFeeService_CalculateIntegerArithmetic(t *testing.T) {
	cfg := &config.Config{
		PlatformFeeBps:   50, // 0.50%
		ForexFeeBps:      25, // 0.25%
		ComplianceFeeBps: 10, // 0.10%
		NetworkFeeBps:    15, // 0.15%
		MinFee:           0,
		MaxFee:           0,
	}

	svc := NewFeeService(cfg)

	// $100.00 = 10000 cents
	breakdown := svc.Calculate(10000)

	assert.Equal(t, int64(50), breakdown.PlatformFee)
	assert.Equal(t, int64(25), breakdown.ForexFee)
	assert.Equal(t, int64(10), breakdown.ComplianceFee)
	assert.Equal(t, int64(15), breakdown.NetworkFee)
	assert.Equal(t, int64(100), breakdown.TotalFee)
}

func TestFeeService_CalculateWithMinFeeClamp(t *testing.T) {
	cfg := &config.Config{
		PlatformFeeBps:   50,
		ForexFeeBps:      25,
		ComplianceFeeBps: 10,
		NetworkFeeBps:    15,
		MinFee:           200, // min fee 200 cents ($2.00)
		MaxFee:           0,
	}

	svc := NewFeeService(cfg)

	breakdown := svc.Calculate(10000)
	assert.Equal(t, int64(200), breakdown.TotalFee)
}
