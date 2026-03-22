package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type CatalogService struct {
	repo *repository.CatalogRepository
}

func NewCatalogService(repo *repository.CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

func (s *CatalogService) GetCategories() ([]domain.Category, error) {
	return s.repo.GetCategories()
}

func (s *CatalogService) GetUnits() ([]domain.Unit, error) {
	return s.repo.GetUnits()
}
