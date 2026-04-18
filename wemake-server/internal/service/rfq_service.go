package service

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

const maxRFQImages = 5

var (
	ErrMaxRFQImages          = errors.New("at most 5 image_urls are allowed")
	ErrInvalidSubCategory    = errors.New("sub_category_id is invalid for the selected category")
	ErrInvalidShippingMethod = errors.New("shipping_method_id is invalid")
)

type RFQService struct {
	repo *repository.RFQRepository
}

func NewRFQService(repo *repository.RFQRepository) *RFQService {
	return &RFQService{repo: repo}
}

func normalizeRFQImageURLs(urls []string) domain.JSONStringArray {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return domain.JSONStringArray(out)
}

func (s *RFQService) Create(rfq *domain.RFQ) error {
	now := time.Now()
	rfq.Title = strings.TrimSpace(rfq.Title)
	rfq.Details = strings.TrimSpace(rfq.Details)
	rfq.Status = "OP"
	rfq.CreatedAt = now
	rfq.UpdatedAt = now
	rfq.UploadedAt = &now

	rfq.ImageURLs = normalizeRFQImageURLs([]string(rfq.ImageURLs))
	if len(rfq.ImageURLs) > maxRFQImages {
		return ErrMaxRFQImages
	}

	if rfq.SubCategoryID != nil {
		valid, err := s.repo.SubCategoryBelongsToCategory(*rfq.SubCategoryID, rfq.CategoryID)
		if err != nil {
			return err
		}
		if !valid {
			return ErrInvalidSubCategory
		}
	}

	if rfq.ShippingMethodID != nil {
		valid, err := s.repo.ShippingMethodExists(*rfq.ShippingMethodID)
		if err != nil {
			return err
		}
		if !valid {
			return ErrInvalidShippingMethod
		}
	}

	return s.repo.Create(rfq)
}

func (s *RFQService) ListByUserID(userID int64, status string) ([]domain.RFQ, error) {
	return s.repo.ListByUserID(userID, strings.TrimSpace(strings.ToUpper(status)))
}

func (s *RFQService) GetByID(userID, rfqID int64) (*domain.RFQ, error) {
	return s.repo.GetByID(userID, rfqID)
}

func (s *RFQService) Cancel(userID, rfqID int64) error {
	return s.repo.Cancel(userID, rfqID)
}

func (s *RFQService) ListMatchingForFactory(factoryID int64, status string) ([]domain.RFQ, error) {
	return s.repo.ListMatchingForFactory(factoryID, strings.TrimSpace(strings.ToUpper(status)))
}

func (s *RFQService) GetForViewer(userID int64, role string, rfqID int64) (*domain.RFQ, error) {
	if role == domain.RoleFactory {
		rfq, err := s.repo.GetByIDAny(rfqID)
		if err != nil {
			return nil, err
		}
		ok, err := s.repo.FactoryHasMatchingCategory(userID, rfq)
		if err != nil {
			return nil, err
		}
		if !ok {
			hasQ, err := s.repo.FactoryHasQuotationOnRFQ(userID, rfqID)
			if err != nil {
				return nil, err
			}
			if !hasQ {
				return nil, sql.ErrNoRows
			}
		}
		return rfq, nil
	}
	return s.GetByID(userID, rfqID)
}
