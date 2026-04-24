package service

import (
	"errors"
	"math"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var ErrCommissionConfigMissing = errors.New("COMMISSION_CONFIG_MISSING")

type Breakdown struct {
	Subtotal                 float64 `json:"subtotal"`
	DiscountAmount           float64 `json:"discount_amount"`
	ShippingCost             float64 `json:"shipping_cost"`
	PackagingCost            float64 `json:"packaging_cost"`
	ToolingMoldCost          float64 `json:"tooling_mold_cost"`
	PreVatBase               float64 `json:"pre_vat_base"`
	VatRate                  float64 `json:"vat_rate"`
	VatAmount                float64 `json:"vat_amount"`
	GrandTotal               float64 `json:"grand_total"`
	PlatformCommissionRate   float64 `json:"platform_commission_rate"`
	PlatformCommissionAmount float64 `json:"platform_commission_amount"`
	FactoryNetReceivable     float64 `json:"factory_net_receivable"`
	PlatformConfigID         int64   `json:"platform_config_id"`
}

type CommissionInput struct {
	Items          []domain.QuotationItem
	DiscountAmount float64
	ShippingCost   float64
	PackagingCost  float64
	ToolingCost    float64
}

type CommissionService struct {
	configs *repository.PlatformConfigRepository
}

func NewCommissionService(configs *repository.PlatformConfigRepository) *CommissionService {
	return &CommissionService{configs: configs}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func resolveEffectiveRate(now time.Time, cfg *domain.PlatformConfig) float64 {
	if cfg.PromoCommissionRate != nil &&
		cfg.PromoStartAt != nil && cfg.PromoEndAt != nil &&
		!now.Before(*cfg.PromoStartAt) && !now.After(*cfg.PromoEndAt) {
		return *cfg.PromoCommissionRate
	}
	return cfg.DefaultCommissionRate
}

func (s *CommissionService) Calculate(in CommissionInput) (*Breakdown, error) {
	cfg, err := s.configs.GetActive()
	if err != nil {
		return nil, ErrCommissionConfigMissing
	}
	var lineSum float64
	for i := range in.Items {
		lineTotal := round2(in.Items[i].Qty * in.Items[i].UnitPrice * (1 - in.Items[i].DiscountPct/100))
		in.Items[i].LineTotal = lineTotal
		lineSum += lineTotal
	}
	subtotal := round2(lineSum - in.DiscountAmount)
	preVatBase := round2(subtotal + in.ShippingCost + in.PackagingCost + in.ToolingCost)
	vatAmount := round2(preVatBase * cfg.VatRate / 100)
	grandTotal := round2(preVatBase + vatAmount)
	commissionRate := resolveEffectiveRate(time.Now().UTC(), cfg)
	commissionAmount := round2(subtotal * commissionRate / 100)
	factoryNet := round2(grandTotal - commissionAmount)

	return &Breakdown{
		Subtotal:                 subtotal,
		DiscountAmount:           round2(in.DiscountAmount),
		ShippingCost:             round2(in.ShippingCost),
		PackagingCost:            round2(in.PackagingCost),
		ToolingMoldCost:          round2(in.ToolingCost),
		PreVatBase:               preVatBase,
		VatRate:                  cfg.VatRate,
		VatAmount:                vatAmount,
		GrandTotal:               grandTotal,
		PlatformCommissionRate:   commissionRate,
		PlatformCommissionAmount: commissionAmount,
		FactoryNetReceivable:     factoryNet,
		PlatformConfigID:         cfg.ConfigID,
	}, nil
}
