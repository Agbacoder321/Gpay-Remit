package services

import (
	"github.com/Gpay-Remit/gpay-remit/config"
)

type FeeBreakdown struct {
	PlatformFee   int64 `json:"platform_fee"`
	ForexFee      int64 `json:"forex_fee"`
	ComplianceFee int64 `json:"compliance_fee"`
	NetworkFee    int64 `json:"network_fee"`
	TotalFee      int64 `json:"total_fee"`
}

type FeeService struct {
	cfg *config.Config
}

func NewFeeService(cfg *config.Config) *FeeService {
	return &FeeService{cfg: cfg}
}

func bps(amount int64, bps int) int64 {
	return (amount * int64(bps)) / 10000
}

// Calculate returns a fee breakdown using integer arithmetic for minor units (cents/stroops).
func (s *FeeService) Calculate(amount int64) FeeBreakdown {
	platform := bps(amount, s.cfg.PlatformFeeBps)
	forex := bps(amount, s.cfg.ForexFeeBps)
	compliance := bps(amount, s.cfg.ComplianceFeeBps)
	network := bps(amount, s.cfg.NetworkFeeBps)

	total := platform + forex + compliance + network

	minFee := int64(s.cfg.MinFee)
	maxFee := int64(s.cfg.MaxFee)

	if minFee > 0 && total < minFee {
		total = minFee
	}
	if maxFee > 0 && total > maxFee {
		total = maxFee
	}

	sum := platform + forex + compliance + network
	if sum > 0 && total != sum {
		platform = (platform * total) / sum
		forex = (forex * total) / sum
		compliance = (compliance * total) / sum
		network = total - (platform + forex + compliance)
	}

	return FeeBreakdown{
		PlatformFee:   platform,
		ForexFee:      forex,
		ComplianceFee: compliance,
		NetworkFee:    network,
		TotalFee:      total,
	}
}
