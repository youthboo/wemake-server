package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type ShowcaseService struct {
	repo *repository.ShowcaseRepository
}

func NewShowcaseService(repo *repository.ShowcaseRepository) *ShowcaseService {
	return &ShowcaseService{repo: repo}
}

func (s *ShowcaseService) ListExplore(contentType string) ([]domain.ShowcaseExploreItem, error) {
	return s.repo.ListExplore(contentType)
}

func (s *ShowcaseService) ListExploreByFactory(factoryID int64, contentType string) ([]domain.ShowcaseExploreItem, error) {
	return s.repo.ListExploreByFactory(factoryID, contentType)
}

func (s *ShowcaseService) Create(showcase *domain.FactoryShowcase) error {
	return s.repo.Create(showcase)
}

func (s *ShowcaseService) GetByID(showcaseID, factoryID int64) (*domain.FactoryShowcase, error) {
	return s.repo.GetByID(showcaseID, factoryID)
}

func (s *ShowcaseService) Update(showcase *domain.FactoryShowcase) error {
	return s.repo.Update(showcase)
}

func (s *ShowcaseService) Delete(showcaseID, factoryID int64) error {
	return s.repo.Delete(showcaseID, factoryID)
}

func (s *ShowcaseService) ListPromoSlides() ([]domain.PromoSlide, error) {
	return s.repo.ListPromoSlides()
}
