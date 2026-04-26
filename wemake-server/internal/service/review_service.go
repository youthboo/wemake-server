package service

import (
	"database/sql"
	"strings"

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

func (s *ReviewService) GetSummaryByFactoryID(factoryID int64) (*domain.FactoryReviewSummary, error) {
	return s.repo.GetSummaryByFactoryID(factoryID)
}

func (s *ReviewService) UpdateByUser(reviewID, userID int64, rating int, comment string) (*domain.FactoryReview, error) {
	if rating < 1 || rating > 5 || strings.TrimSpace(comment) == "" || len(strings.TrimSpace(comment)) > 2000 {
		return nil, sql.ErrNoRows
	}
	return s.repo.UpdateByUser(reviewID, userID, rating, comment)
}

func (s *ReviewService) DeleteByUser(reviewID, userID int64) error {
	item, err := s.repo.SoftDeleteByUser(reviewID, userID)
	if err != nil {
		return err
	}
	tx, err := s.repo.DB().Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.repo.SyncFactoryAggregateTx(tx, item.FactoryID); err != nil {
		return err
	}
	return tx.Commit()
}
