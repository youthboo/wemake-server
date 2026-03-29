package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type ReviewService struct {
	repo *repository.ReviewRepository
}

func NewReviewService(repo *repository.ReviewRepository) *ReviewService {
	return &ReviewService{repo: repo}
}

func (s *ReviewService) ListByFactoryID(factoryID int64) ([]domain.FactoryReview, error) {
	return s.repo.ListByFactoryID(factoryID)
}

func (s *ReviewService) Create(review *domain.FactoryReview) error {
	return s.repo.Create(review)
}
