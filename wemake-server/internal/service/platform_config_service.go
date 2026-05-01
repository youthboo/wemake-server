package service

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var (
	ErrPlatformConfigValidation = errors.New("invalid platform config")
	ErrPlatformConfigNotFound   = errors.New("platform config not found")
	ErrPlatformDefaultDelete    = errors.New("cannot delete default platform config")
	ErrPlatformConfigInUse      = errors.New("platform config is in use")
)

type PlatformConfigService struct {
	db    *sqlx.DB
	repo  *repository.PlatformConfigRepository
	audit *repository.AdminAuditRepository
}

func NewPlatformConfigService(db *sqlx.DB, repo *repository.PlatformConfigRepository, audit *repository.AdminAuditRepository) *PlatformConfigService {
	return &PlatformConfigService{db: db, repo: repo, audit: audit}
}

func (s *PlatformConfigService) GetActive() (*domain.PlatformConfig, error) {
	return s.repo.GetActive()
}

func (s *PlatformConfigService) ListHistory() ([]domain.PlatformConfig, error) {
	return s.repo.ListHistory()
}

func (s *PlatformConfigService) ListAll() ([]domain.PlatformConfig, error) {
	return s.repo.ListAll()
}

func (s *PlatformConfigService) CreateVersion(cfg *domain.PlatformConfig) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if cfg.CurrencyCode == "" {
		cfg.CurrencyCode = "THB"
	}
	cfg.CurrencyCode = strings.ToUpper(strings.TrimSpace(cfg.CurrencyCode))
	cfg.EffectiveFrom = time.Now().UTC()
	if err := s.repo.CloseActive(tx); err != nil {
		return err
	}
	if err := s.repo.Create(tx, cfg); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PlatformConfigService) CreateConfig(req domain.CreatePlatformConfigRequest, actorID int64, ip *string) (*domain.PlatformConfig, error) {
	label := strings.TrimSpace(req.Label)
	if !validLabel(label) || !validRate(req.DefaultCommissionRate) {
		return nil, ErrPlatformConfigValidation
	}
	vatRate := 7.0
	if req.VatRate != nil {
		if !validRate(*req.VatRate) {
			return nil, ErrPlatformConfigValidation
		}
		vatRate = *req.VatRate
	}
	currency := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if currency == "" {
		currency = "THB"
	}
	var effectiveTo *time.Time
	if req.EffectiveTo != nil && strings.TrimSpace(*req.EffectiveTo) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.EffectiveTo))
		if err != nil || !parsed.After(time.Now().UTC()) {
			return nil, ErrPlatformConfigValidation
		}
		effectiveTo = &parsed
	}
	cfg := &domain.PlatformConfig{
		Label:                 &label,
		DefaultCommissionRate: req.DefaultCommissionRate,
		VatRate:               vatRate,
		CurrencyCode:          currency,
		EffectiveTo:           effectiveTo,
		CreatedBy:             &actorID,
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.repo.CreatePackage(tx, cfg); err != nil {
		return nil, err
	}
	if err := s.insertAudit(actorID, "PLATFORM_CONFIG_CREATE", "platform_config", cfg.ConfigID, cfg, ip); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *PlatformConfigService) UpdateConfig(configID int64, req domain.UpdatePlatformConfigRequest, actorID int64, ip *string) (*domain.PlatformConfig, error) {
	label := strings.TrimSpace(req.Label)
	if configID <= 0 || !validLabel(label) || !validRate(req.DefaultCommissionRate) || !validRate(req.VatRate) {
		return nil, ErrPlatformConfigValidation
	}
	before, err := s.repo.GetByID(configID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return nil, ErrPlatformConfigNotFound
		}
		return nil, err
	}
	after := &domain.PlatformConfig{
		Label:                 &label,
		DefaultCommissionRate: req.DefaultCommissionRate,
		VatRate:               req.VatRate,
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.repo.UpdateConfig(tx, configID, after); err != nil {
		if repository.IsNotFoundError(err) {
			return nil, ErrPlatformConfigNotFound
		}
		return nil, err
	}
	payload := map[string]interface{}{"before": before, "after": after}
	if err := s.insertAudit(actorID, "PLATFORM_CONFIG_UPDATE", "platform_config", configID, payload, ip); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (s *PlatformConfigService) DeleteConfig(configID int64, actorID int64, ip *string) error {
	if configID <= 0 {
		return ErrPlatformConfigValidation
	}
	target, err := s.repo.GetByID(configID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return ErrPlatformConfigNotFound
		}
		return err
	}
	defaultCfg, err := s.repo.GetDefault()
	if err != nil {
		return err
	}
	if target.ConfigID == defaultCfg.ConfigID {
		return ErrPlatformDefaultDelete
	}
	count, err := s.repo.CountFactoriesUsingConfig(configID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrPlatformConfigInUse
	}
	if err := s.repo.DeleteConfig(configID); err != nil {
		if repository.IsNotFoundError(err) {
			return ErrPlatformConfigNotFound
		}
		return err
	}
	return s.insertAudit(actorID, "PLATFORM_CONFIG_DELETE", "platform_config", configID, target, ip)
}

func (s *PlatformConfigService) FactoriesUsingConfig(configID int64) (int, error) {
	return s.repo.CountFactoriesUsingConfig(configID)
}

func validLabel(label string) bool {
	return len([]rune(label)) >= 2
}

func validRate(rate float64) bool {
	return rate >= 0 && rate <= 100
}

func (s *PlatformConfigService) insertAudit(actorID int64, action, targetType string, targetID int64, payload interface{}, ip *string) error {
	if s.audit == nil {
		return nil
	}
	raw, _ := json.Marshal(payload)
	return s.audit.Insert(&domain.AdminAuditLog{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   strconv.FormatInt(targetID, 10),
		Payload:    raw,
		IPAddress:  ip,
		CreatedAt:  time.Now().UTC(),
	})
}
