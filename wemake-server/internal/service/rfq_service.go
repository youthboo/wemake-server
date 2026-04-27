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
	ErrMaxRFQReferenceImages = errors.New("at most 5 reference_images are allowed")
	ErrInvalidSubCategory    = errors.New("sub_category_id is invalid for the selected category")
	ErrInvalidShippingMethod = errors.New("shipping_method_id is invalid")
	ErrRFQDetailsRequired    = errors.New("description/details must not be empty")
	ErrRFQInspectionInvalid  = errors.New("inspection_type is invalid")
)

type RFQService struct {
	repo          *repository.RFQRepository
	factoryRepo   *repository.FactoryRepository
	notifications *NotificationService
}

func NewRFQService(repo *repository.RFQRepository, factoryRepo *repository.FactoryRepository, notifications *NotificationService) *RFQService {
	return &RFQService{repo: repo, factoryRepo: factoryRepo, notifications: notifications}
}

func normalizeStringSlice(values []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (s *RFQService) Create(rfq *domain.RFQ) error {
	now := time.Now()
	rfq.Title = strings.TrimSpace(rfq.Title)
	rfq.Details = strings.TrimSpace(rfq.Details)
	rfq.Status = "OP"
	rfq.CreatedAt = now
	rfq.UpdatedAt = now
	rfq.UploadedAt = &now

	rfq.ReferenceImages = normalizeStringSlice([]string(rfq.ReferenceImages))
	if len(rfq.ReferenceImages) > maxRFQImages {
		return ErrMaxRFQReferenceImages
	}
	if rfq.Details == "" {
		return ErrRFQDetailsRequired
	}
	if !rfq.SampleRequired {
		rfq.SampleQty = nil
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
	if err := s.repo.Create(rfq); err != nil {
		return err
	}
	s.notifyMatchingFactories(rfq)
	return nil
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
	if s.factoryRepo != nil {
		approvalStatus, err := s.factoryRepo.GetApprovalStatus(factoryID)
		if err != nil {
			return nil, err
		}
		if approvalStatus == "SU" {
			return []domain.RFQ{}, nil
		}
	}
	return s.repo.ListMatchingForFactory(factoryID, strings.TrimSpace(strings.ToUpper(status)))
}

func (s *RFQService) GetForViewer(userID int64, role string, rfqID int64) (*domain.RFQ, error) {
	if role == domain.RoleFactory {
		// Any approved (non-suspended) factory may view any RFQ regardless of category match.
		if s.factoryRepo != nil {
			approvalStatus, err := s.factoryRepo.GetApprovalStatus(userID)
			if err != nil {
				return nil, err
			}
			if approvalStatus == "SU" {
				return nil, sql.ErrNoRows
			}
		}
		return s.repo.GetByIDAny(rfqID)
	}
	return s.GetByID(userID, rfqID)
}

func validateRFQEnums(rfq *domain.RFQ) error {
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
	rfq.Details = strings.TrimSpace(rfq.Details)
	rfq.ReferenceImages = normalizeStringSlice([]string(rfq.ReferenceImages))
	if len(rfq.ReferenceImages) > maxRFQImages {
		return ErrMaxRFQReferenceImages
	}
	if rfq.Details == "" {
		return ErrRFQDetailsRequired
	}
	if !rfq.SampleRequired {
		rfq.SampleQty = nil
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

func (s *RFQService) notifyMatchingFactories(rfq *domain.RFQ) {
	if s.notifications == nil || s.repo == nil || rfq == nil || rfq.RFQID <= 0 {
		return
	}
	factoryIDs, err := s.repo.ListMatchingFactoryIDs(rfq)
	if err != nil {
		return
	}
	title := "RFQ ใหม่ตรงหมวด"
	rfqTitle := strings.TrimSpace(rfq.Title)
	if rfqTitle == "" {
		rfqTitle = "RFQ ใหม่"
	}
	for _, factoryID := range factoryIDs {
		createNotificationSafe(s.notifications, &domain.Notification{
			UserID:  factoryID,
			Type:    "RFQ_RECEIVED",
			Title:   title,
			Message: "มี RFQ ใหม่ที่ตรงหมวดของคุณ: " + rfqTitle,
			LinkTo:  rfqLink(rfq.RFQID),
			Data: notificationData(map[string]interface{}{
				"rfq_id":    rfq.RFQID,
				"rfq_title": rfqTitle,
				"url":       rfqLink(rfq.RFQID),
			}),
			ReferenceID: &rfq.RFQID,
			CreatedAt:   rfq.CreatedAt,
		})
	}
}
