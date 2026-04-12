package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type CertificateService struct {
	repo *repository.CertificateRepository
}

func NewCertificateService(repo *repository.CertificateRepository) *CertificateService {
	return &CertificateService{repo: repo}
}

func (s *CertificateService) ListByFactoryID(factoryID int64) ([]domain.FactoryCertificate, error) {
	return s.repo.ListByFactoryID(factoryID)
}

func (s *CertificateService) Create(cert *domain.FactoryCertificate) error {
	return s.repo.Create(cert)
}

func (s *CertificateService) DeleteByMapID(factoryID, mapID int64) error {
	return s.repo.DeleteByMapID(factoryID, mapID)
}
