package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type FactoryService struct {
	repo *repository.FactoryRepository
}

func NewFactoryService(repo *repository.FactoryRepository) *FactoryService {
	return &FactoryService{repo: repo}
}

func (s *FactoryService) ListPublic() ([]domain.FactoryListItem, error) {
	return s.repo.ListPublicVerified()
}

func (s *FactoryService) GetPublicDetail(factoryID int64) (*domain.FactoryPublicDetail, error) {
	return s.repo.GetPublicDetail(factoryID)
}

func (s *FactoryService) FactoryExistsActive(factoryID int64) (bool, error) {
	return s.repo.FactoryExistsActive(factoryID)
}

func (s *FactoryService) ListFactoryCategories(factoryID int64) ([]domain.FactoryProfileCategory, error) {
	return s.repo.ListFactoryCategories(factoryID)
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
