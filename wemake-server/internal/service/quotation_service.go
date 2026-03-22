package service

import (
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type QuotationService struct {
	repo *repository.QuotationRepository
}

func NewQuotationService(repo *repository.QuotationRepository) *QuotationService {
	return &QuotationService{repo: repo}
}

func (s *QuotationService) Create(item *domain.Quotation) error {
	now := time.Now()
	item.Status = "PD"
	item.CreateTime = now
	item.LogTimestamp = now
	return s.repo.Create(item)
}

func (s *QuotationService) ListByRFQID(rfqID int64) ([]domain.Quotation, error) {
	return s.repo.ListByRFQID(rfqID)
}

func (s *QuotationService) GetByID(quotationID int64) (*domain.Quotation, error) {
	return s.repo.GetByID(quotationID)
}

func (s *QuotationService) UpdateStatus(quotationID int64, status string) error {
	return s.repo.UpdateStatus(quotationID, strings.TrimSpace(strings.ToUpper(status)))
}
