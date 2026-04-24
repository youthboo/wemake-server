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
	ErrMaxRFQImages           = errors.New("at most 5 image_urls are allowed")
	ErrInvalidSubCategory     = errors.New("sub_category_id is invalid for the selected category")
	ErrInvalidShippingMethod  = errors.New("shipping_method_id is invalid")
	ErrRFQIncotermsInvalid    = errors.New("incoterms is invalid")
	ErrRFQPaymentTermsInvalid = errors.New("payment_terms is invalid")
	ErrRFQInspectionInvalid   = errors.New("inspection_type is invalid")
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
	if err := validateRFQEnums(rfq); err != nil {
		return err
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

func validateRFQEnums(rfq *domain.RFQ) error {
	if rfq.Incoterms != nil {
		switch strings.TrimSpace(strings.ToUpper(*rfq.Incoterms)) {
		case "EXW", "FOB", "CIF", "DDP":
		default:
			return ErrRFQIncotermsInvalid
		}
	}
	if rfq.PaymentTerms != nil {
		switch strings.TrimSpace(*rfq.PaymentTerms) {
		case "50_50", "30_70", "net_30", "lc_at_sight":
		default:
			return ErrRFQPaymentTermsInvalid
		}
	}
	if rfq.InspectionType != nil {
		switch strings.TrimSpace(*rfq.InspectionType) {
		case "self", "third_party", "buyer_onsite":
		default:
			return ErrRFQInspectionInvalid
		}
	}
	return nil
}

func (s *RFQService) Patch(userID, rfqID int64, rfq *domain.RFQ) error {
	existing, err := s.repo.GetByID(userID, rfqID)
	if err != nil {
		return err
	}
	if existing.Status != "OP" {
		return errors.New("rfq is not editable")
	}
	rfq.RFQID = rfqID
	rfq.UserID = userID
	rfq.Status = existing.Status
	rfq.CreatedAt = existing.CreatedAt
	rfq.UploadedAt = existing.UploadedAt
	rfq.UpdatedAt = time.Now()
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
	if err := validateRFQEnums(rfq); err != nil {
		return err
	}
	return s.repo.Patch(userID, rfqID, rfq)
}
