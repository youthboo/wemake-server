package service

import (
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type ProductionService struct {
	repo *repository.ProductionRepository
}

func NewProductionService(repo *repository.ProductionRepository) *ProductionService {
	return &ProductionService{repo: repo}
}

func (s *ProductionService) Create(item *domain.ProductionUpdate) error {
	item.Description = strings.TrimSpace(item.Description)
	item.ImageURL = strings.TrimSpace(item.ImageURL)
	item.CreatedAt = time.Now()
	return s.repo.Create(item)
}

func (s *ProductionService) ListByOrderID(orderID int64) ([]domain.ProductionUpdate, error) {
	return s.repo.ListByOrderID(orderID)
}

func (s *ProductionService) Patch(updateID int64, description *string, imageURL *string) error {
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		description = &trimmed
	}
	if imageURL != nil {
		trimmed := strings.TrimSpace(*imageURL)
		imageURL = &trimmed
	}
	return s.repo.Patch(updateID, description, imageURL)
}
