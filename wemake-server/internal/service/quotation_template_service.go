package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type QuotationTemplateService struct {
	repo *repository.QuotationTemplateRepository
}

func NewQuotationTemplateService(repo *repository.QuotationTemplateRepository) *QuotationTemplateService {
	return &QuotationTemplateService{repo: repo}
}

func (s *QuotationTemplateService) ListByFactoryID(factoryID int64) ([]domain.QuotationTemplate, error) {
	return s.repo.ListByFactoryID(factoryID)
}

func (s *QuotationTemplateService) Create(t *domain.QuotationTemplate) error {
	t.IsActive = true
	return s.repo.Create(t)
}

func (s *QuotationTemplateService) Update(t *domain.QuotationTemplate) error {
	return s.repo.Update(t)
}

func (s *QuotationTemplateService) Delete(templateID, factoryID int64) error {
	return s.repo.Delete(templateID, factoryID)
}
