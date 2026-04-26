package service

import (
	"database/sql"
	"strings"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

const maxReviewImages = 5

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
	review.ImageURLs = normalizeReviewImageURLs(review.ImageURLs)
	if len(review.ImageURLs) > maxReviewImages {
		return ErrReviewImagesInvalid
	}
	return s.repo.Create(review)
}

func (s *ReviewService) GetSummaryByFactoryID(factoryID int64) (*domain.FactoryReviewSummary, error) {
	return s.repo.GetSummaryByFactoryID(factoryID)
}

func (s *ReviewService) UpdateByUser(reviewID, userID int64, rating int, comment string, imageURLs domain.StringArray) (*domain.FactoryReview, error) {
	if rating < 1 || rating > 5 || strings.TrimSpace(comment) == "" || len(strings.TrimSpace(comment)) > 2000 {
		return nil, sql.ErrNoRows
	}
	imageURLs = normalizeReviewImageURLs(imageURLs)
	if len(imageURLs) > maxReviewImages {
		return nil, ErrReviewImagesInvalid
	}
	return s.repo.UpdateByUser(reviewID, userID, rating, comment, imageURLs)
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

func normalizeReviewImageURLs(values domain.StringArray) domain.StringArray {
	seen := make(map[string]struct{})
	out := make(domain.StringArray, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
