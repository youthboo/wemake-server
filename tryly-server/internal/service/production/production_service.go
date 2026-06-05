package production

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	domainstatus "github.com/yourusername/wemake/internal/domain/status"
	"github.com/yourusername/wemake/internal/helper"
	productionrepo "github.com/yourusername/wemake/internal/repository/production"
	"github.com/yourusername/wemake/internal/domainutil"
)

var (
	ErrProductionNotOrderFactory        = errors.New("NOT_ORDER_FACTORY")
	ErrProductionNotOrderCustomer       = errors.New("NOT_ORDER_CUSTOMER")
	ErrProductionOrderLocked            = errors.New("ORDER_LOCKED")
	ErrProductionAnotherStepInProgress  = errors.New("ANOTHER_STEP_IN_PROGRESS")
	ErrProductionInvalidStateTransition = errors.New("INVALID_STATE_TRANSITION")
	ErrProductionDownstreamInFlight     = errors.New("DOWNSTREAM_IN_FLIGHT")
	ErrProductionStepOrderViolation     = errors.New("STEP_ORDER_VIOLATION")
	ErrProductionInsufficientEvidence   = errors.New("INSUFFICIENT_EVIDENCE")
	ErrProductionPaymentConfirmRequired = errors.New("PAYMENT_CONFIRMATION_REQUIRED")
	ErrProductionInvalidStep            = errors.New("INVALID_STEP")
	ErrProductionStepIDRequired         = errors.New("STEP_ID_REQUIRED")
	ErrProductionInvalidStatus          = errors.New("INVALID_STATUS")
	ErrProductionInvalidImageURL        = errors.New("INVALID_IMAGE_URL")
	ErrProductionInvalidImageFormat     = errors.New("INVALID_IMAGE_FORMAT")
	ErrProductionMaxImages              = errors.New("MAX_5_IMAGES")
	ErrProductionDescriptionTooLong     = errors.New("DESCRIPTION_TOO_LONG")
	ErrProductionReasonRequired         = errors.New("REASON_REQUIRED")
	ErrProductionStepNotFound           = errors.New("STEP_NOT_FOUND")
)

type ProductionRuleError struct {
	Err     error
	Details map[string]interface{}
}

func (e *ProductionRuleError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ProductionRuleError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type ProductionService struct {
	repo *productionrepo.ProductionRepository
}

type ProductionWriteInput struct {
	StepID                 int64
	Status                 string
	Description            string
	ImageURLs              []string
	ConfirmPaymentTrigger  bool
	HeaderPaymentConfirmed bool
	// step_id=4: บันทึก tracking_no / courier ลง orders
	TrackingNo string
	Courier    string
}

func NewProductionService(repo *productionrepo.ProductionRepository) *ProductionService {
	return &ProductionService{repo: repo}
}

func (s *ProductionService) ListSteps(factoryTypeID *int64) ([]domain.ProductionStepTemplate, error) {
	return s.repo.ListActiveSteps()
}

func (s *ProductionService) ListByOrderID(orderID, userID int64) (*domain.ProductionUpdatesList, error) {
	order, role, err := s.repo.LoadAuthorizedOrder(orderID, userID)
	if err != nil {
		return nil, err
	}
	if !s.repo.IsAdminRole(role) && !(s.repo.IsFactoryRole(role) && order.FactoryID == userID) && !(s.repo.IsCustomerRole(role) && order.UserID == userID) {
		if s.repo.IsFactoryRole(role) {
			return nil, &ProductionRuleError{Err: ErrProductionNotOrderFactory}
		}
		return nil, &ProductionRuleError{Err: ErrProductionNotOrderCustomer}
	}
	steps, err := s.repo.ListActiveSteps()
	if err != nil {
		return nil, err
	}
	status := domainstatus.NormalizeOrder(order.OrderStatus)
	if domainstatus.IsProductionReadLockedOrder(status) {
		lockReason := domainstatus.ProductionLockReason(status)
		return &domain.ProductionUpdatesList{
			OrderID:          orderID,
			Updates:          []domain.ProductionUpdate{},
			OrderStatus:      status,
			ProductionLocked: true,
			LockReason:       lockReason,
			LockContext:      buildProductionLockContext(order, lockReason),
			TemplatePreview:  steps,
		}, nil
	}
	persisted, err := s.repo.ListByOrderID(orderID)
	if err != nil {
		return nil, err
	}
	return &domain.ProductionUpdatesList{
		OrderID:          orderID,
		Updates:          s.repo.InflateUpdates(orderID, steps, persisted),
		OrderStatus:      status,
		ProductionLocked: false,
		TemplatePreview:  steps,
	}, nil
}

func (s *ProductionService) Upsert(orderID, userID int64, input ProductionWriteInput) (*domain.ProductionUpdateResult, error) {
	input.Status = domainutil.NormalizeStatus(input.Status)
	input.Description = strings.TrimSpace(input.Description)
	// step_id=0 = "ยืนยันรับงาน" อนุญาต, reject เฉพาะค่า negative
	if input.StepID < 0 {
		return nil, &ProductionRuleError{Err: ErrProductionStepIDRequired}
	}
	if len(input.Description) > 2000 {
		return nil, &ProductionRuleError{Err: ErrProductionDescriptionTooLong}
	}
	if input.Status != "IP" && input.Status != "CD" {
		return nil, &ProductionRuleError{Err: ErrProductionInvalidStatus}
	}
	if err := validateImageURLs(input.ImageURLs); err != nil {
		return nil, err
	}

	var result *domain.ProductionUpdateResult
	if err := helper.WithTxOptions(context.Background(), s.repo.DB(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *sqlx.Tx) error {
		role, err := s.repo.GetUserRole(userID)
		if err != nil {
			return err
		}
		role = normalizeUserRole(role)
		if !s.repo.IsFactoryRole(role) {
			return &ProductionRuleError{Err: ErrProductionNotOrderFactory}
		}

		order, err := s.repo.GetOrderForUpdateTx(tx, orderID)
		if err != nil {
			return err
		}
		if order.FactoryID != userID {
			return &ProductionRuleError{Err: ErrProductionNotOrderFactory}
		}
		if domainstatus.IsProductionLockedOrder(order.OrderStatus) {
			return &ProductionRuleError{Err: ErrProductionOrderLocked}
		}

		steps, err := s.repo.ListActiveStepsTx(tx)
		if err != nil {
			return err
		}
		step := s.repo.StepByID(steps, input.StepID)
		if step == nil || !step.IsActive {
			return &ProductionRuleError{Err: ErrProductionInvalidStep}
		}

		persisted, err := s.repo.ListByOrderIDTx(tx, orderID)
		if err != nil {
			return err
		}
		inflated := s.repo.InflateUpdates(orderID, steps, persisted)
		current := s.repo.GetUpdateByOrderAndStep(orderID, input.StepID, inflated)
		if current == nil {
			current = &domain.ProductionUpdate{
				OrderID:    orderID,
				StepID:     input.StepID,
				StepCode:   step.StepCode,
				StepNameTH: step.StepNameTH,
				StepNameEN: step.StepNameEN,
				SortOrder:  step.SortOrder,
				Status:     "PD",
				ImageURLs:  domain.StringArray{},
			}
		}

		// step_id=0 (ยืนยันรับงาน) ต้องทำก่อน step ทุกตัว → เปลี่ยน guard จาก > 1 เป็น > 0
		if step.SortOrder > 0 {
			prevStep := s.repo.StepBySortOrder(steps, step.SortOrder-1)
			prev := s.repo.GetUpdateByOrderAndStep(orderID, prevStep.StepID, inflated)
			// Block both IP and CD if the previous step hasn't been completed
			if (input.Status == "IP" || input.Status == "CD") && (prev == nil || prev.Status != "CD") {
				return &ProductionRuleError{
					Err:     ErrProductionStepOrderViolation,
					Details: map[string]interface{}{"required_previous_step": prevStep.StepCode},
				}
			}
		}

		// Cannot revert a completed step back to in-progress
		if current.Status == "CD" && input.Status == "IP" {
			return &ProductionRuleError{Err: ErrProductionInvalidStateTransition}
		}
		// Cannot mark a rejected step as done without re-submitting as IP first
		if current.Status == "RJ" && input.Status == "CD" {
			return &ProductionRuleError{Err: ErrProductionInvalidStateTransition}
		}
		// PD → CD is allowed: factory can complete a step in one click (with evidence)

		if active := s.repo.GetActiveInProgressStep(inflated, input.StepID); input.Status == "IP" && active != nil && active.Status == "IP" && active.StepID != input.StepID {
			return &ProductionRuleError{
				Err:     ErrProductionAnotherStepInProgress,
				Details: map[string]interface{}{"in_progress_step_id": active.StepID},
			}
		}

		if input.Status == "CD" {
			// step_id=0 = ยืนยันรับงาน — ไม่ต้องแนบรูป
			requiredPhotos := 1
			if input.StepID == 0 {
				requiredPhotos = 0
			}
			if len(input.ImageURLs) < requiredPhotos {
				return &ProductionRuleError{
					Err:     ErrProductionInsufficientEvidence,
					Details: map[string]interface{}{"required": requiredPhotos, "provided": len(input.ImageURLs)},
				}
			}
			if step.IsPaymentTrigger && (!input.ConfirmPaymentTrigger || !input.HeaderPaymentConfirmed) {
				return &ProductionRuleError{Err: ErrProductionPaymentConfirmRequired}
			}
		}

		updatedBy := userID
		update := &domain.ProductionUpdate{
			OrderID:         orderID,
			StepID:          input.StepID,
			StepCode:        step.StepCode,
			StepNameTH:      step.StepNameTH,
			StepNameEN:      step.StepNameEN,
			SortOrder:       step.SortOrder,
			Status:          input.Status,
			Description:     input.Description,
			ImageURLs:       domain.StringArray(input.ImageURLs),
			UpdatedByUserID: &updatedBy,
		}
		if input.Status == "CD" {
			now := time.Now().UTC()
			update.CompletedAt = &now
		}

		// step_id=4: บันทึก tracking_no / courier ทันทีที่มีค่า (ก่อน early-return เพื่อให้ update ได้แม้ submit ซ้ำ)
		if input.StepID == 4 {
			trackingVal := strings.TrimSpace(input.TrackingNo)
			courierVal := strings.TrimSpace(input.Courier)
			if trackingVal != "" || courierVal != "" {
				if _, err := tx.Exec(
					`UPDATE orders SET tracking_no = NULLIF($1,''), courier = NULLIF($2,''), updated_at = NOW() WHERE order_id = $3`,
					trackingVal, courierVal, orderID,
				); err != nil {
					return err
				}
			}
		}

		if current.Status == input.Status && current.Description == input.Description && equalStringArrays(current.ImageURLs, update.ImageURLs) {
			update.UpdateID = current.UpdateID
			update.CreatedAt = current.CreatedAt
			update.LastUpdatedAt = current.LastUpdatedAt
			update.CompletedAt = current.CompletedAt
			update.RejectedReason = current.RejectedReason
			result = &domain.ProductionUpdateResult{
				Update:      *current,
				OrderStatus: order.OrderStatus,
			}
			return nil
		}

		if err := s.repo.UpsertTx(tx, update); err != nil {
			return err
		}

		var autoProgressed *domain.ProductionUpdateAutoProgressed
		if input.Status == "CD" {
			if step.IsPaymentTrigger {
				if err := s.repo.InsertDomainEventTx(tx, "production.payment_triggered", map[string]interface{}{
					"order_id":     orderID,
					"step_code":    step.StepCode,
					"update_id":    update.UpdateID,
					"triggered_at": time.Now().UTC(),
				}); err != nil {
					return err
				}
			}

			if nextStep := s.repo.StepBySortOrder(steps, step.SortOrder+1); nextStep != nil && nextStep.IsActive {
				nextUpdatedBy := userID
				next := &domain.ProductionUpdate{
					OrderID:         orderID,
					StepID:          nextStep.StepID,
					StepCode:        nextStep.StepCode,
					StepNameTH:      nextStep.StepNameTH,
					StepNameEN:      nextStep.StepNameEN,
					SortOrder:       nextStep.SortOrder,
					Status:          "IP",
					Description:     "",
					ImageURLs:       domain.StringArray{},
					UpdatedByUserID: &nextUpdatedBy,
				}
				if err := s.repo.UpsertTx(tx, next); err != nil {
					return err
				}
				autoProgressed = &domain.ProductionUpdateAutoProgressed{StepID: nextStep.StepID, Status: "IP"}
			}
		}

		newOrderStatus := order.OrderStatus
		// step_id=0 (ยืนยันรับงาน): CD จาก PD → PR
		if input.StepID == 0 && input.Status == "CD" && domainstatus.NormalizeOrder(order.OrderStatus) == domain.OrderStatusPaymentDone {
			newOrderStatus = domain.OrderStatusProduction
		}
		// step 3 CD → QC, step 4 CD → SH, step 5 CD → CP
		if input.Status == "CD" {
			switch input.StepID {
			case 3:
				newOrderStatus = domain.OrderStatusQualityCheck
			case 4:
				newOrderStatus = domain.OrderStatusShipping
			case 5:
				newOrderStatus = domain.OrderStatusComplete
			}
		}
		if newOrderStatus != order.OrderStatus {
			if _, err := tx.Exec(`UPDATE orders SET status = $1, updated_at = NOW() WHERE order_id = $2`, newOrderStatus, orderID); err != nil {
				return err
			}
		}
		result = &domain.ProductionUpdateResult{
			Update:         *update,
			OrderStatus:    newOrderStatus,
			AutoProgressed: autoProgressed,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ProductionService) Reject(updateID, userID int64, reason string) (*domain.ProductionUpdate, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 1000 {
		return nil, &ProductionRuleError{Err: ErrProductionReasonRequired}
	}

	var item *domain.ProductionUpdate
	if err := helper.WithTxOptions(context.Background(), s.repo.DB(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *sqlx.Tx) error {
		role, err := s.repo.GetUserRole(userID)
		if err != nil {
			return err
		}
		role = normalizeUserRole(role)

		updateCtx, err := s.repo.GetUpdateByIDForUpdateTx(tx, updateID)
		if err != nil {
			return err
		}
		order, err := s.repo.GetOrderForUpdateTx(tx, updateCtx.OrderID)
		if err != nil {
			return err
		}
		if domainstatus.IsProductionLockedOrder(order.OrderStatus) {
			return &ProductionRuleError{Err: ErrProductionOrderLocked}
		}

		if s.repo.IsCustomerRole(role) {
			if order.UserID != userID {
				return &ProductionRuleError{Err: ErrProductionNotOrderCustomer}
			}
		} else if !s.repo.IsAdminRole(role) {
			return &ProductionRuleError{Err: ErrProductionNotOrderCustomer}
		}

		if updateCtx.Status != "CD" {
			return &ProductionRuleError{Err: ErrProductionInvalidStateTransition}
		}
		persisted, err := s.repo.ListByOrderIDTx(tx, updateCtx.OrderID)
		if err != nil {
			return err
		}
		if s.repo.HasDownstreamInFlight(persisted, updateCtx.SortOrder) {
			return &ProductionRuleError{Err: ErrProductionDownstreamInFlight}
		}
		item, err = s.repo.RejectTx(tx, updateID, reason, userID)
		if err != nil {
			return err
		}
		item.StepCode = updateCtx.StepCode
		item.StepNameTH = updateCtx.StepNameTH
		item.StepNameEN = updateCtx.StepNameEN
		item.SortOrder = updateCtx.SortOrder
		return nil
	}); err != nil {
		return nil, err
	}
	return item, nil
}

func validateImageURLs(items []string) error {
	if len(items) > 5 {
		return &ProductionRuleError{
			Err:     ErrProductionMaxImages,
			Details: map[string]interface{}{"max": 5, "provided": len(items)},
		}
	}
	seen := make(map[string]struct{}, len(items))
	for _, raw := range items {
		v := strings.TrimSpace(raw)
		if v == "" || len(v) > 2048 {
			return &ProductionRuleError{Err: ErrProductionInvalidImageFormat}
		}
		u, err := url.Parse(v)
		scheme := strings.ToLower(u.Scheme)
		if err != nil || (scheme != "https" && scheme != "http") || u.Host == "" {
			return &ProductionRuleError{Err: ErrProductionInvalidImageURL}
		}
		if _, ok := seen[v]; ok {
			return &ProductionRuleError{Err: ErrProductionInvalidImageFormat}
		}
		seen[v] = struct{}{}
	}
	return nil
}

func normalizeUserRole(role string) string {
	switch domainutil.NormalizeStatus(role) {
	case "CT":
		return "CU"
	default:
		return domainutil.NormalizeStatus(role)
	}
}

func equalStringArrays(a, b domain.StringArray) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func buildProductionLockContext(order *productionrepo.ProductionOrderContext, reason string) map[string]interface{} {
	depositAmount := helper.RoundCurrency(orderDepositAmountFallback(order))
	depositPercent := helper.PercentOf(depositAmount, order.TotalAmount)
	switch reason {
	case "PENDING_DEPOSIT":
		dueDate := depositDueDateForProduction(order)
		return map[string]interface{}{
			"deposit_amount":   depositAmount,
			"deposit_currency": "THB",
			"deposit_due_date": dueDate,
			"deposit_percent":  depositPercent,
			"payment_url":      fmt.Sprintf("/orders/%d/payment?stage=deposit", order.OrderID),
		}
	case "DEPOSIT_EXPIRED":
		expiredAt := depositDueDateForProduction(order)
		return map[string]interface{}{
			"deposit_amount":      depositAmount,
			"deposit_currency":    "THB",
			"expired_at":          expiredAt,
			"contact_factory_url": fmt.Sprintf("/chat?factory_id=%d&order_id=%d", order.FactoryID, order.OrderID),
		}
	case "ORDER_CANCELLED":
		cancelledAt := order.CreatedAt.In(helper.ThailandLocation)
		return map[string]interface{}{
			"cancelled_at":       cancelledAt,
			"cancelled_by_actor": "SYSTEM",
			"refund_status":      "NOT_APPLICABLE",
			"refund_amount":      0.0,
		}
	default:
		return map[string]interface{}{"support_url": "/support"}
	}
}

func depositDueDateForProduction(order *productionrepo.ProductionOrderContext) time.Time {
	// ใช้ due_date จาก payment_schedules ถ้ามี (ตรงกับ order_service.lookupDepositDueDate)
	if order.DepositDueDate != nil {
		d := order.DepositDueDate.In(helper.ThailandLocation)
		return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, helper.ThailandLocation)
	}
	// fallback: created_at + 3 วัน
	due := order.CreatedAt.In(helper.ThailandLocation).AddDate(0, 0, 3)
	return time.Date(due.Year(), due.Month(), due.Day(), 23, 59, 59, 0, helper.ThailandLocation)
}

func orderDepositAmountFallback(order *productionrepo.ProductionOrderContext) float64 {
	if order.DepositAmount > 0 {
		return order.DepositAmount
	}
	return helper.RoundCurrency(order.TotalAmount * 0.3)
}

func AsProductionRuleError(err error) (*ProductionRuleError, bool) {
	var target *ProductionRuleError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func RuleErrorWithDetails(err error, details map[string]interface{}) error {
	return &ProductionRuleError{Err: err, Details: details}
}

func WrapRuleError(err error, details map[string]interface{}) *ProductionRuleError {
	return &ProductionRuleError{Err: err, Details: details}
}

func UnexpectedTransition(current, next string) error {
	return &ProductionRuleError{
		Err: ErrProductionInvalidStateTransition,
		Details: map[string]interface{}{
			"current_status": current,
			"requested":      next,
		},
	}
}

func BuildStepOrderViolation(requiredStepCode string) error {
	return &ProductionRuleError{
		Err: ErrProductionStepOrderViolation,
		Details: map[string]interface{}{
			"required_previous_step": requiredStepCode,
		},
	}
}

func BuildInProgressViolation(stepID int64) error {
	return &ProductionRuleError{
		Err: ErrProductionAnotherStepInProgress,
		Details: map[string]interface{}{
			"in_progress_step_id": stepID,
		},
	}
}

func BuildEvidenceViolation(required, provided int64) error {
	return &ProductionRuleError{
		Err: ErrProductionInsufficientEvidence,
		Details: map[string]interface{}{
			"required": required,
			"provided": provided,
		},
	}
}

func ExplainProductionError(err error) string {
	switch {
	case errors.Is(err, ErrProductionNotOrderFactory):
		return "factory caller does not own the order"
	case errors.Is(err, ErrProductionNotOrderCustomer):
		return "customer caller does not own the order"
	case errors.Is(err, ErrProductionOrderLocked):
		return "order is locked"
	case errors.Is(err, ErrProductionAnotherStepInProgress):
		return "another step is already in progress"
	case errors.Is(err, ErrProductionInvalidStateTransition):
		return "invalid state transition"
	case errors.Is(err, ErrProductionDownstreamInFlight):
		return "downstream steps are already in progress"
	case errors.Is(err, ErrProductionStepOrderViolation):
		return "previous step must be completed first"
	case errors.Is(err, ErrProductionInsufficientEvidence):
		return "insufficient evidence to complete step"
	case errors.Is(err, ErrProductionPaymentConfirmRequired):
		return "payment confirmation required"
	case errors.Is(err, ErrProductionInvalidStep):
		return "invalid production step"
	case errors.Is(err, ErrProductionStepIDRequired):
		return "step_id is required"
	case errors.Is(err, ErrProductionInvalidStatus):
		return "invalid production update status"
	case errors.Is(err, ErrProductionMaxImages):
		return "too many image urls"
	case errors.Is(err, ErrProductionInvalidImageFormat):
		return "invalid image url format"
	case errors.Is(err, ErrProductionInvalidImageURL):
		return "invalid image urls"
	case errors.Is(err, ErrProductionDescriptionTooLong):
		return "description too long"
	case errors.Is(err, ErrProductionReasonRequired):
		return "rejection reason required"
	default:
		return fmt.Sprintf("unexpected production error: %v", err)
	}
}
