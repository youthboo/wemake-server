package factory

import (
	"github.com/yourusername/wemake/internal/domain"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
)

// ErrFactoryProfileExists is exported from the repository for handler use.
var ErrFactoryProfileExists = factoryrepo.ErrFactoryProfileExists

type FactoryService struct {
	repo *factoryrepo.FactoryRepository
}

func NewFactoryService(repo *factoryrepo.FactoryRepository) *FactoryService {
	return &FactoryService{repo: repo}
}

func (s *FactoryService) ListPublic(scope ...string) ([]domain.FactoryListItem, error) {
	return s.repo.ListPublicVerified(scope...)
}

func (s *FactoryService) GetPublicDetail(factoryID int64) (*domain.FactoryPublicDetail, error) {
	return s.repo.GetPublicDetail(factoryID)
}

func (s *FactoryService) CreateProfile(userID int64, factoryName string, taxID string, provinceID *int64, categoryIDs []int64, subCategoryIDs []int64, certID int64, documentURL string, certNumber string, certExpireDate string) error {
	return s.repo.CreateProfile(userID, factoryName, taxID, provinceID, categoryIDs, subCategoryIDs, certID, documentURL, certNumber, certExpireDate)
}

func (s *FactoryService) PatchProfile(factoryID int64, fields map[string]interface{}) error {
	return s.repo.PatchProfile(factoryID, fields)
}

func (s *FactoryService) FactoryExistsActive(factoryID int64) (bool, error) {
	return s.repo.FactoryExistsActive(factoryID)
}

func (s *FactoryService) ListFactoryCategories(factoryID int64) ([]domain.FactoryProfileCategory, error) {
	return s.repo.ListFactoryCategories(factoryID)
}

func (s *FactoryService) ListFactoryCategoryIDs(factoryID int64) ([]int64, error) {
	ids, err := s.repo.ListFactoryCategoryIDs(factoryID)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		return []int64{}, nil
	}
	return ids, nil
}

func (s *FactoryService) AddFactoryCategory(factoryID, categoryID int64) error {
	return s.repo.AddFactoryCategory(factoryID, categoryID)
}

func (s *FactoryService) RemoveFactoryCategory(factoryID, categoryID int64) error {
	return s.repo.RemoveFactoryCategory(factoryID, categoryID)
}

func (s *FactoryService) ReplaceFactoryCategories(factoryID int64, categoryIDs []int64) error {
	return s.repo.ReplaceFactoryCategories(factoryID, categoryIDs)
}

func (s *FactoryService) ListFactorySubCategories(factoryID int64) ([]domain.FactoryProfileSubCategory, error) {
	return s.repo.ListFactorySubCategories(factoryID)
}

func (s *FactoryService) AddFactorySubCategory(factoryID, subCategoryID int64) error {
	return s.repo.AddFactorySubCategory(factoryID, subCategoryID)
}

func (s *FactoryService) RemoveFactorySubCategory(factoryID, subCategoryID int64) error {
	return s.repo.RemoveFactorySubCategory(factoryID, subCategoryID)
}

func (s *FactoryService) ReplaceFactorySubCategories(factoryID int64, subCategoryIDs []int64) error {
	return s.repo.ReplaceFactorySubCategories(factoryID, subCategoryIDs)
}

func (s *FactoryService) GetDashboard(factoryID int64) (*domain.FactoryDashboard, error) {
	return s.repo.GetDashboard(factoryID)
}

func (s *FactoryService) GetAnalytics(factoryID int64) (*domain.FactoryAnalytics, error) {
	return s.repo.GetAnalytics(factoryID)
}

func (s *FactoryService) GetPortal(factoryID int64) (*domain.FactoryPortal, error) {
	return s.repo.GetPortal(factoryID)
}
