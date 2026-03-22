package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type FactoryService struct {
	repo *repository.FactoryRepository
}

func NewFactoryService(repo *repository.FactoryRepository) *FactoryService {
	return &FactoryService{repo: repo}
}

func (s *FactoryService) CreateFactory(name, email, phone, address, description string) (*domain.Factory, error) {
	factory := &domain.Factory{
		ID:          uuid.New().String(),
		Name:        name,
		Email:       email,
		Phone:       phone,
		Address:     address,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateFactory(factory); err != nil {
		return nil, err
	}

	return factory, nil
}

func (s *FactoryService) GetFactoryByID(id string) (*domain.Factory, error) {
	return s.repo.GetFactoryByID(id)
}

func (s *FactoryService) GetAllFactories() ([]domain.Factory, error) {
	return s.repo.GetAllFactories()
}

func (s *FactoryService) UpdateFactory(id, name, email, phone, address, description string) (*domain.Factory, error) {
	factory, err := s.repo.GetFactoryByID(id)
	if err != nil {
		return nil, err
	}

	factory.Name = name
	factory.Email = email
	factory.Phone = phone
	factory.Address = address
	factory.Description = description
	factory.UpdatedAt = time.Now()

	if err := s.repo.UpdateFactory(factory); err != nil {
		return nil, err
	}

	return factory, nil
}

func (s *FactoryService) DeleteFactory(id string) error {
	return s.repo.DeleteFactory(id)
}
