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

func (s *ShowcaseService) GetShowcasesByFactory(factoryID int64, contentType string, callerID int64) ([]domain.ShowcaseByFactoryItem, error) {
	return s.repo.GetShowcasesByFactory(factoryID, contentType, callerID)
}

func (s *ShowcaseService) GetDetail(showcaseID int64) (*domain.ShowcaseDetail, error) {
	return s.repo.GetDetail(showcaseID)
}

func (s *ShowcaseService) Create(showcase *domain.FactoryShowcase) error {
	return s.repo.Create(showcase)
}

func (s *ShowcaseService) GetByID(showcaseID, factoryID int64) (*domain.FactoryShowcase, error) {
	return s.repo.GetByID(showcaseID, factoryID)
}

func (s *ShowcaseService) GetAnalytics(showcaseID, factoryID int64) (*domain.ShowcaseAnalytics, error) {
	return s.repo.GetAnalytics(showcaseID, factoryID)
}

func (s *ShowcaseService) Update(showcase *domain.FactoryShowcase) error {
	return s.repo.Update(showcase)
}

func (s *ShowcaseService) Delete(showcaseID, factoryID int64) error {
	return s.repo.Delete(showcaseID, factoryID)
}

func (s *ShowcaseService) RecordView(showcaseID int64) error {
	return s.repo.IncrementViewCount(showcaseID)
}

func (s *ShowcaseService) ListPromoSlides() ([]domain.PromoSlide, error) {
	return s.repo.ListPromoSlides()
}

func (s *ShowcaseService) CreateImage(img *domain.ShowcaseImage, factoryID int64) error {
	return s.repo.CreateImage(img, factoryID)
}

func (s *ShowcaseService) DeleteImage(showcaseID, imageID, factoryID int64) error {
	return s.repo.DeleteImage(showcaseID, imageID, factoryID)
}

func (s *ShowcaseService) GetSections(showcaseID, factoryID int64) ([]domain.ShowcaseSection, error) {
	return s.repo.GetSections(showcaseID, factoryID)
}

func (s *ShowcaseService) BulkReplaceSections(showcaseID, factoryID int64, inputs []domain.ShowcaseSectionInput) error {
	return s.repo.BulkReplaceSections(showcaseID, factoryID, inputs)
}

func (s *ShowcaseService) GetSpecs(showcaseID, factoryID int64) ([]domain.ShowcaseSpec, error) {
	return s.repo.GetSpecs(showcaseID, factoryID)
}

func (s *ShowcaseService) BulkReplaceSpecs(showcaseID, factoryID int64, inputs []domain.ShowcaseSpecInput) error {
	return s.repo.BulkReplaceSpecs(showcaseID, factoryID, inputs)
}

func (s *ShowcaseService) PatchImage(showcaseID, imageID, factoryID int64, sortOrder *int, caption *string) (*domain.ShowcaseImage, error) {
	return s.repo.PatchImage(showcaseID, imageID, factoryID, sortOrder, caption)
}

func (s *ShowcaseService) DeleteSection(showcaseID, sectionID, factoryID int64) error {
	return s.repo.DeleteSection(showcaseID, sectionID, factoryID)
}
