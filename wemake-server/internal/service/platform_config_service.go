package service

import (
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type PlatformConfigService struct {
	db   *sqlx.DB
	repo *repository.PlatformConfigRepository
}

func NewPlatformConfigService(db *sqlx.DB, repo *repository.PlatformConfigRepository) *PlatformConfigService {
	return &PlatformConfigService{db: db, repo: repo}
}

func (s *PlatformConfigService) GetActive() (*domain.PlatformConfig, error) {
	return s.repo.GetActive()
}

func (s *PlatformConfigService) ListHistory() ([]domain.PlatformConfig, error) {
	return s.repo.ListHistory()
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
