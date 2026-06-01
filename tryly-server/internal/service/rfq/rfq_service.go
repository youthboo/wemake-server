package rfq

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
	domainstatus "github.com/yourusername/wemake/internal/domain/status"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
	rfqrepo "github.com/yourusername/wemake/internal/repository/rfq"
	notificationservice "github.com/yourusername/wemake/internal/service/notification"
)

const maxRFQImages = 5

var (
	ErrMaxRFQReferenceImages = errors.New("at most 5 reference_images are allowed")
	ErrInvalidSubCategory    = errors.New("sub_category_id is invalid for the selected category")
	ErrInvalidCategory       = errors.New("category_id is invalid")
	ErrInvalidShippingMethod = errors.New("shipping_method_id is invalid")
	ErrRFQDetailsRequired    = errors.New("description/details must not be empty")
	ErrRFQDetailsTooShort    = errors.New("sample request details must be at least 20 characters")
	ErrRFQKindInvalid        = errors.New("request_kind must be PR, PS, MS, or MR")
	ErrRFQSampleQtyInvalid   = errors.New("sample request quantity is outside allowed range")
	ErrRFQWrongScope         = errors.New("WRONG_SCOPE")
	ErrHasActiveQuotation    = errors.New("HAS_ACTIVE_QUOTATION")
	ErrQuotationAccepted     = errors.New("QUOTATION_ACCEPTED")
	ErrRFQNotEditable        = errors.New("RFQ_NOT_EDITABLE")
)

type RFQService struct {
	repo          *rfqrepo.RFQRepository
	factoryRepo   *factoryrepo.FactoryRepository
	notifications *notificationservice.NotificationService
}

func NewRFQService(repo *rfqrepo.RFQRepository, factoryRepo *factoryrepo.FactoryRepository, notifications *notificationservice.NotificationService) *RFQService {
	return &RFQService{repo: repo, factoryRepo: factoryRepo, notifications: notifications}
}

func (s *RFQService) Create(rfq *domain.RFQ) error {
	now := time.Now()
	rfq.Title = strings.TrimSpace(rfq.Title)
	rfq.Details = strings.TrimSpace(rfq.Details)
	rfq.RequestKind = normalizeRFQKind(rfq.RequestKind)
	if rfq.RequestKind == "" {
		return ErrRFQKindInvalid
	}
	if err := s.validateCategoryScope(rfq.RequestKind, rfq.CategoryID); err != nil {
		return err
	}
	if err := validateRFQKindRules(rfq); err != nil {
		return err
	}
	rfq.Status = "OP"
	rfq.CreatedAt = now
	rfq.UpdatedAt = now
	rfq.UploadedAt = &now

	rfq.ReferenceImages = pq.StringArray(domainutil.NormalizeStringSlice([]string(rfq.ReferenceImages)))
	if len(rfq.ReferenceImages) > maxRFQImages {
		return ErrMaxRFQReferenceImages
	}
	if rfq.Details == "" {
		return ErrRFQDetailsRequired
	}

	if rfq.RequestKind != domain.RequestKindMaterialSample && rfq.RequestKind != domain.RequestKindRawMaterial && rfq.SubCategoryID != nil {
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
	if rfq.Targeting != "specific" {
		rfq.Targeting = "all"
	}

	if rfq.Targeting == "specific" && len(rfq.TargetFactoryIDs) > 0 {
		// Use a transaction so the RFQ row and target-factory rows land atomically.
		if err := helper.WithTx(nil, s.repo.DB(), func(tx *sqlx.Tx) error {
			if err := s.repo.CreateTx(tx, rfq); err != nil {
				return err
			}
			return s.repo.InsertTargetFactoriesTx(tx, rfq.RFQID, rfq.TargetFactoryIDs)
		}); err != nil {
			return err
		}
	} else {
		if err := s.repo.Create(rfq); err != nil {
			return err
		}
	}

	s.notifyMatchingFactories(rfq)
	return nil
}

func (s *RFQService) ListByUserID(userID int64, status string) ([]domain.RFQ, error) {
	return s.repo.ListByUserID(userID, domainstatus.NormalizeCode(status))
}

func (s *RFQService) GetByID(userID, rfqID int64) (*domain.RFQ, error) {
	return s.repo.GetByID(userID, rfqID)
}

func (s *RFQService) Cancel(userID, rfqID int64) error {
	return s.repo.Cancel(userID, rfqID)
}

// Close lets the customer manually stop accepting new quotations for an open RFQ (OP → CL).
// Unlike Cancel, existing accepted quotations/orders remain untouched.
func (s *RFQService) Close(userID, rfqID int64) error {
	return s.repo.CloseRFQ(userID, rfqID)
}

func (s *RFQService) ListMatchingForFactory(factoryID int64, status string, kind string, showDismissed bool) ([]domain.RFQ, error) {
	if s.factoryRepo != nil {
		approvalStatus, err := s.factoryRepo.GetApprovalStatus(factoryID)
		if err != nil {
			return nil, err
		}
		if approvalStatus == "SU" {
			return []domain.RFQ{}, nil
		}
	}
	normalizedKind := domainutil.NormalizeStatus(kind)
	if normalizedKind != "" && normalizeRFQKind(normalizedKind) == "" {
		return nil, ErrRFQKindInvalid
	}
	return s.repo.ListMatchingForFactory(factoryID, domainstatus.NormalizeCode(status), normalizedKind, showDismissed)
}

func (s *RFQService) GetFactoryBoard(factoryID int64, status, kind string, showDismissed bool) (*domain.FactoryRFQBoardResponse, error) {
	rfqs, err := s.ListMatchingForFactory(factoryID, status, kind, showDismissed)
	if err != nil {
		return nil, err
	}
	if rfqs == nil {
		rfqs = []domain.RFQ{}
	}

	catIDs, err := s.factoryRepo.ListFactoryCategoryIDs(factoryID)
	if err != nil {
		return nil, err
	}
	if catIDs == nil {
		catIDs = []int64{}
	}

	return &domain.FactoryRFQBoardResponse{
		RFQs:               rfqs,
		FactoryCategoryIDs: catIDs,
	}, nil
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
		return nil, ErrInvalidCategory
	}
	var categoryExists bool
	if err := s.repo.DB().Get(&categoryExists, `SELECT EXISTS(SELECT 1 FROM lbi_categories WHERE category_id = $1)`, categoryID); err != nil {
		return nil, err
	}
	if !categoryExists {
		return nil, ErrInvalidCategory
	}
	if err := s.validateCategoryScope(normalizedKind, categoryID); err != nil {
		return nil, err
	}
	if normalizedKind == domain.RequestKindMaterialSample || normalizedKind == domain.RequestKindRawMaterial {
		subCategoryID = nil
	}
	if normalizedKind != domain.RequestKindMaterialSample && normalizedKind != domain.RequestKindRawMaterial && subCategoryID != nil {
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
		rfq, err := s.repo.GetByIDAny(rfqID)
		if err != nil {
			return nil, err
		}
		return rfq, nil
	}
	return s.GetByID(userID, rfqID)
}

func (s *RFQService) DismissRFQ(factoryID, rfqID int64) (*domain.FactoryRFQDismissal, bool, error) {
	exists, err := s.repo.RFQExists(rfqID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, sql.ErrNoRows
	}
	status, hasQuotation, err := s.repo.FactoryQuotationStatus(factoryID, rfqID)
	if err != nil {
		return nil, false, err
	}
	if hasQuotation {
		switch status {
		case "AC":
			return nil, false, ErrQuotationAccepted
		case "PD":
			return nil, false, ErrHasActiveQuotation
		}
	}
	return s.repo.DismissFactoryRFQ(factoryID, rfqID)
}

func (s *RFQService) UndismissRFQ(factoryID, rfqID int64) error {
	exists, err := s.repo.RFQExists(rfqID)
	if err != nil {
		return err
	}
	if !exists {
		return sql.ErrNoRows
	}
	return s.repo.UndismissFactoryRFQ(factoryID, rfqID)
}

func (s *RFQService) Patch(userID, rfqID int64, rfq *domain.RFQ) error {
	existing, err := s.repo.GetByID(userID, rfqID)
	if err != nil {
		return err
	}
	if existing.Status != "OP" {
		return ErrRFQNotEditable
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
	if err := s.validateCategoryScope(rfq.RequestKind, rfq.CategoryID); err != nil {
		return err
	}
	rfq.CreatedAt = existing.CreatedAt
	rfq.UploadedAt = existing.UploadedAt
	rfq.UpdatedAt = time.Now()
	rfq.Details = strings.TrimSpace(rfq.Details)
	rfq.ReferenceImages = pq.StringArray(domainutil.NormalizeStringSlice([]string(rfq.ReferenceImages)))
	if rfq.CertificationsRequired == nil {
		rfq.CertificationsRequired = existing.CertificationsRequired
	}
	if rfq.CertificationsRequired == nil {
		rfq.CertificationsRequired = pq.StringArray{}
	}
	if len(rfq.ReferenceImages) > maxRFQImages {
		return ErrMaxRFQReferenceImages
	}
	if rfq.Details == "" {
		return ErrRFQDetailsRequired
	}
	if err := validateRFQKindRules(rfq); err != nil {
		return err
	}
	if rfq.RequestKind != domain.RequestKindMaterialSample && rfq.RequestKind != domain.RequestKindRawMaterial && rfq.SubCategoryID != nil {
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
	return s.repo.Patch(userID, rfqID, rfq)
}

func normalizeRFQKind(kind string) string {
	switch domainutil.NormalizeStatus(kind) {
	case "":
		return domain.RequestKindProduction
	case domain.RequestKindProduction, domain.RequestKindProductSample, domain.RequestKindMaterialSample, domain.RequestKindRawMaterial:
		return domainutil.NormalizeStatus(kind)
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
		if len([]rune(strings.TrimSpace(rfq.Details))) < 20 {
			return ErrRFQDetailsTooShort
		}
		// ใช้ค่า target_price ที่ลูกค้าส่งมา ถ้าไม่มีค่อย default เป็น 0
		if rfq.TargetPrice == nil {
			zero := helper.ZeroMoney()
			rfq.TargetPrice = &zero
		}
	case domain.RequestKindMaterialSample:
		if rfq.CategoryID <= 0 {
			return ErrInvalidSubCategory
		}
		if rfq.Quantity < 1 || rfq.Quantity > 5 {
			return ErrRFQSampleQtyInvalid
		}
		if len([]rune(strings.TrimSpace(rfq.Details))) < 20 {
			return ErrRFQDetailsTooShort
		}
		// ใช้ค่า target_price ที่ลูกค้าส่งมา ถ้าไม่มีค่อย default เป็น 0
		if rfq.TargetPrice == nil {
			zero := helper.ZeroMoney()
			rfq.TargetPrice = &zero
		}
	case domain.RequestKindProduction, domain.RequestKindRawMaterial:
		return nil
	default:
		return ErrRFQKindInvalid
	}
	return nil
}

func (s *RFQService) validateCategoryScope(kind string, categoryID int64) error {
	scope, found, err := s.repo.CategoryScope(categoryID)
	if err != nil {
		return err
	}
	if !found {
		return ErrInvalidCategory
	}
	expected := "PD"
	if kind == domain.RequestKindMaterialSample || kind == domain.RequestKindRawMaterial {
		expected = "MT"
	}
	if scope != expected {
		return ErrRFQWrongScope
	}
	return nil
}

// UpdateTargets replaces the target factory list for an RFQ owned by userID.
func (s *RFQService) UpdateTargets(userID, rfqID int64, factoryIDs []int64) error {
	err := s.repo.UpdateTargets(userID, rfqID, factoryIDs)
	if err != nil && err.Error() == "RFQ_NOT_EDITABLE" {
		return ErrRFQNotEditable
	}
	return err
}

func (s *RFQService) notifyMatchingFactories(rfq *domain.RFQ) {
	if s.notifications == nil || s.repo == nil || rfq == nil || rfq.RFQID <= 0 {
		return
	}
	var factoryIDs []int64
	if rfq.Targeting == "specific" {
		// Only notify the explicitly-chosen factories.
		factoryIDs = rfq.TargetFactoryIDs
	} else {
		var err error
		factoryIDs, err = s.repo.ListMatchingFactoryIDs(rfq)
		if err != nil {
			return
		}
	}
	title := "RFQ ใหม่ตรงหมวด"
	rfqTitle := strings.TrimSpace(rfq.Title)
	if rfqTitle == "" {
		rfqTitle = "RFQ ใหม่"
	}
	for _, factoryID := range factoryIDs {
		helper.CreateNotificationSafe(s.notifications, &domain.Notification{
			UserID:  factoryID,
			Type:    "RFQ_RECEIVED",
			Title:   title,
			Message: "มี RFQ ใหม่ที่ตรงหมวดของคุณ: " + rfqTitle,
			LinkTo:  helper.FactoryRFQLink(rfq.RFQID),
			Data: helper.NotificationData(map[string]interface{}{
				"rfq_id":    rfq.RFQID,
				"rfq_title": rfqTitle,
				"url":       helper.FactoryRFQLink(rfq.RFQID),
			}),
			ReferenceID: &rfq.RFQID,
			CreatedAt:   rfq.CreatedAt,
		})
	}
}
