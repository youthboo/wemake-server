package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type MasterService struct {
	repo *repository.MasterRepository
}

func NewMasterService(repo *repository.MasterRepository) *MasterService {
	return &MasterService{repo: repo}
}

func (s *MasterService) GetProvinces() ([]domain.LBIProvince, error) {
	return s.repo.GetProvinces()
}

func (s *MasterService) GetDistricts(provinceID *int64) ([]domain.LBIDistrict, error) {
	return s.repo.GetDistricts(provinceID)
}

func (s *MasterService) GetSubDistricts(districtID *int64) ([]domain.LBISubDistrict, error) {
	return s.repo.GetSubDistricts(districtID)
}

func (s *MasterService) GetFactoryTypes() ([]domain.LBIFactoryType, error) {
	return s.repo.GetFactoryTypes()
}

func (s *MasterService) GetProductCategories(parentID *int64) ([]domain.LBIProductCategory, error) {
	return s.repo.GetProductCategories(parentID)
}

func (s *MasterService) GetProductionSteps(factoryTypeID *int64) ([]domain.LBIProduction, error) {
	return s.repo.GetProductionSteps(factoryTypeID)
}

func (s *MasterService) GetUnits() ([]domain.LBIUnit, error) {
	return s.repo.GetUnits()
}

func (s *MasterService) GetShippingMethods() ([]domain.LBIShippingMethod, error) {
	return s.repo.GetShippingMethods()
}

func (s *MasterService) GetCertificates() ([]domain.LBIMasterCertificate, error) {
	return s.repo.GetCertificates()
}
