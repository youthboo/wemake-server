package factory

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/logger"
	catalogservice "github.com/yourusername/wemake/internal/service/catalog"
	masterservice "github.com/yourusername/wemake/internal/service/master"
	userservice "github.com/yourusername/wemake/internal/service/user"
)

type ProfileInitService struct {
	factory *FactoryService
	master  *masterservice.MasterService
	catalog *catalogservice.CatalogService
	address *userservice.AddressService
}

func NewProfileInitService(
	factory *FactoryService,
	master *masterservice.MasterService,
	catalog *catalogservice.CatalogService,
	address *userservice.AddressService,
) *ProfileInitService {
	return &ProfileInitService{
		factory: factory,
		master:  master,
		catalog: catalog,
		address: address,
	}
}

func (s *ProfileInitService) GetProfileInit(userID int64) (*domain.ProfileInitResponse, error) {
	logger.Debug("building factory profile init", "user_id", userID)

	factory, err := s.factory.GetPublicDetail(userID)
	if err != nil {
		logger.Error("GetPublicDetail failed", "user_id", userID, "err", err)
		return nil, err
	}

	factoryTypes, err := s.master.GetFactoryTypes()
	if err != nil {
		logger.Error("GetFactoryTypes failed", "user_id", userID, "err", err)
		return nil, err
	}
	if factoryTypes == nil {
		factoryTypes = []domain.LBIFactoryType{}
	}

	lbiCategories, err := s.catalog.GetCategories(domain.CatalogScopeAll, 0)
	if err != nil {
		logger.Error("GetCategories failed", "user_id", userID, "err", err)
		return nil, err
	}
	if lbiCategories == nil {
		lbiCategories = []domain.Category{}
	}

	addresses, err := s.address.ListByUserID(userID)
	if err != nil {
		logger.Error("ListByUserID failed", "user_id", userID, "err", err)
		return nil, err
	}
	if addresses == nil {
		addresses = []domain.Address{}
	}

	certTypes, err := s.master.GetCertificates()
	if err != nil {
		logger.Error("GetCertificates failed", "user_id", userID, "err", err)
		return nil, err
	}
	if certTypes == nil {
		certTypes = []domain.LBIMasterCertificate{}
	}

	subCategories := make([]domain.SubCategory, 0)
	if factory != nil {
		for _, cat := range factory.Categories {
			subs, subErr := s.catalog.GetSubCategories(int64(cat.CategoryID))
			if subErr != nil {
				logger.Warn("profile init sub-categories query failed, continuing", "user_id", userID, "category_id", cat.CategoryID, "err", subErr)
				continue
			}
			subCategories = append(subCategories, subs...)
		}
	}

	return &domain.ProfileInitResponse{
		Factory:          factory,
		FactoryTypes:     factoryTypes,
		LBICategories:    lbiCategories,
		Addresses:        addresses,
		CertificateTypes: certTypes,
		SubCategories:    subCategories,
	}, nil
}
