package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type AddressService struct {
	repo *repository.AddressRepository
}

func NewAddressService(repo *repository.AddressRepository) *AddressService {
	return &AddressService{repo: repo}
}

func (s *AddressService) ListByUserID(userID int64) ([]domain.Address, error) {
	return s.repo.ListByUserID(userID)
}

func (s *AddressService) Create(address *domain.Address) error {
	return s.repo.Create(address)
}

func (s *AddressService) Patch(userID, addressID int64, fields map[string]interface{}) error {
	return s.repo.Patch(userID, addressID, fields)
}
