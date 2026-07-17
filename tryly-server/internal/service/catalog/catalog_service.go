package catalog

import (
	"sort"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/logger"
	catalogrepo "github.com/yourusername/wemake/internal/repository/catalog"
	frontendrepo "github.com/yourusername/wemake/internal/repository/frontend"
	showcaseservice "github.com/yourusername/wemake/internal/service/showcase"
)

var exploreShowcaseTypes = []string{"PD", "MT", "PM", "ID"}

type CatalogService struct {
	repo            *catalogrepo.CatalogRepository
	showcaseService *showcaseservice.ShowcaseService
	frontendRepo    *frontendrepo.FrontendRepository
}

func NewCatalogService(repo *catalogrepo.CatalogRepository, showcaseService *showcaseservice.ShowcaseService, frontendRepo *frontendrepo.FrontendRepository) *CatalogService {
	return &CatalogService{repo: repo, showcaseService: showcaseService, frontendRepo: frontendRepo}
}

func (s *CatalogService) GetExplore() (*domain.ExploreResponse, error) {
	logger.Debug("building explore page data")

	categories, err := s.GetCategories(domain.CatalogScopeAll, 6)
	if err != nil {
		return nil, err
	}
	if categories == nil {
		categories = []domain.Category{}
	}

	showcases, err := s.showcaseService.GetHomeShowcases(exploreShowcaseTypes, 8)
	if err != nil {
		return nil, err
	}

	// โรงงานแนะนำ: verified ก่อน → rating สูงก่อน → ตัดสูงสุด 8 ตัว
	exploreFactories := make([]domain.ExploreFactory, 0)
	if factoryRows, fErr := s.frontendRepo.ListFactories(); fErr == nil {
		sort.Slice(factoryRows, func(i, j int) bool {
			if factoryRows[i].Verified != factoryRows[j].Verified {
				return factoryRows[i].Verified
			}
			return factoryRows[i].Rating > factoryRows[j].Rating
		})
		if len(factoryRows) > 8 {
			factoryRows = factoryRows[:8]
		}
		for _, f := range factoryRows {
			exploreFactories = append(exploreFactories, domain.ExploreFactory{
				ID:       f.ID,
				Name:     f.Name,
				Image:    f.ImageURL.String,
				Location: f.Location.String,
				Rating:   f.Rating,
				Reviews:  f.ReviewCount,
				Verified: f.Verified,
			})
		}
	} else {
		logger.Warn("explore: failed to fetch factories, returning empty", "err", fErr)
	}

	return &domain.ExploreResponse{
		Categories: categories,
		Showcases:  showcases,
		Factories:  exploreFactories,
	}, nil
}

func (s *CatalogService) GetCategories(scope string, limit int) ([]domain.Category, error) {
	return s.repo.GetCategories(scope, limit)
}

func (s *CatalogService) GetSubCategories(categoryID int64) ([]domain.SubCategory, error) {
	return s.repo.GetSubCategories(categoryID)
}

func (s *CatalogService) GetAllSubCategories(scope string) ([]domain.SubCategory, error) {
	return s.repo.GetAllSubCategories(scope)
}

func (s *CatalogService) GetCategoriesWithSubs(scope string, limit int) ([]domain.CategoryWithSubs, error) {
	return s.repo.GetCategoriesWithSubs(scope, limit)
}

func (s *CatalogService) GetCategoriesByHub(hubID int64) ([]domain.Category, error) {
	return s.repo.GetCategoriesByHub(hubID)
}

func (s *CatalogService) GetHubs(scope string) ([]domain.HubWithCategories, error) {
	return s.repo.GetHubs(scope)
}

func (s *CatalogService) GetUnits() ([]domain.Unit, error) {
	return s.repo.GetUnits()
}

func (s *CatalogService) GetHubShowcases(hubID int64, limitPerHub int) ([]domain.HubShowcaseGroup, error) {
	return s.showcaseService.GetHubShowcases(hubID, limitPerHub)
}
