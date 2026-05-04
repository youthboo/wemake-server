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
	ErrRFQKindInvalid        = errors.New("request_kind must be PR, PS, or MS")
	ErrRFQSampleQtyInvalid   = errors.New("sample request quantity is outside allowed range")
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
	rfq.RequestKind = normalizeRFQKind(rfq.RequestKind)
	if rfq.RequestKind == "" {
		return ErrRFQKindInvalid
	}
	if err := validateRFQKindRules(rfq); err != nil {
		return err
	}
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

func (s *RFQService) ListMatchingForFactory(factoryID int64, status string, kind string) ([]domain.RFQ, error) {
	if s.factoryRepo != nil {
		approvalStatus, err := s.factoryRepo.GetApprovalStatus(factoryID)
		if err != nil {
			return nil, err
		}
		if approvalStatus == "SU" {
			return []domain.RFQ{}, nil
		}
	}
	normalizedKind := strings.TrimSpace(strings.ToUpper(kind))
	if normalizedKind != "" && normalizeRFQKind(normalizedKind) == "" {
		return nil, ErrRFQKindInvalid
	}
	return s.repo.ListMatchingForFactory(factoryID, strings.TrimSpace(strings.ToUpper(status)), normalizedKind)
}

type PreviewFactoriesResult struct {
	Kind          string  `json:"kind"`
	CategoryID    int64   `json:"category_id"`
	SubCategoryID *int64  `json:"sub_category_id,omitempty"`
	MatchCount    int     `json:"match_count"`
	FactoryIDs    []int64 `json:"factory_ids,omitempty"`
}

func (s *RFQService) PreviewFactories(kind string, categoryID int64, subCategoryID *int64) (*PreviewFactoriesResult, error) {
	normalizedKind := normalizeRFQKind(kind)
	if normalizedKind == "" {
		return nil, ErrRFQKindInvalid
	}
	if categoryID <= 0 {
		return nil, ErrInvalidSubCategory
	}
	if subCategoryID != nil {
		valid, err := s.repo.SubCategoryBelongsToCategory(*subCategoryID, categoryID)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidSubCategory
		}
	}
	ids, err := s.repo.ListMatchingFactoryIDsForKind(normalizedKind, categoryID, subCategoryID)
	if err != nil {
		return nil, err
	}
	return &PreviewFactoriesResult{
		Kind:          normalizedKind,
		CategoryID:    categoryID,
		SubCategoryID: subCategoryID,
		MatchCount:    len(ids),
		FactoryIDs:    ids,
	}, nil
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
	rfq.RequestKind = normalizeRFQKind(rfq.RequestKind)
	if rfq.RequestKind == "" {
		rfq.RequestKind = existing.RequestKind
	}
	if rfq.RequestKind == "" {
		rfq.RequestKind = domain.RequestKindProduction
	}
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
	if err := validateRFQKindRules(rfq); err != nil {
		return err
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

func normalizeRFQKind(kind string) string {
	switch strings.TrimSpace(strings.ToUpper(kind)) {
	case "":
		return domain.RequestKindProduction
	case domain.RequestKindProduction, domain.RequestKindProductSample, domain.RequestKindMaterialSample:
		return strings.TrimSpace(strings.ToUpper(kind))
	default:
		return ""
	}
}

func validateRFQKindRules(rfq *domain.RFQ) error {
	if rfq == nil {
		return nil
	}
	switch rfq.RequestKind {
	case domain.RequestKindProductSample:
		if rfq.Quantity < 1 || rfq.Quantity > 10 {
			return ErrRFQSampleQtyInvalid
		}
		zero := float64(0)
		rfq.TargetUnitPrice = &zero
		rfq.SampleRequired = true
	case domain.RequestKindMaterialSample:
		if rfq.CategoryID <= 0 {
			return ErrInvalidSubCategory
		}
		if rfq.Quantity < 1 || rfq.Quantity > 5 {
			return ErrRFQSampleQtyInvalid
		}
		zero := float64(0)
		rfq.TargetUnitPrice = &zero
		rfq.SampleRequired = true
	case domain.RequestKindProduction:
		return nil
	default:
		return ErrRFQKindInvalid
	}
	return nil
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
