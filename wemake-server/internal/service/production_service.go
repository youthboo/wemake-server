package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
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
	ErrProductionInvalidImageURL        = errors.New("INVALID_IMAGE_URL")
	ErrProductionDescriptionTooLong     = errors.New("DESCRIPTION_TOO_LONG")
	ErrProductionReasonRequired         = errors.New("REASON_REQUIRED")
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
	repo *repository.ProductionRepository
}

type ProductionWriteInput struct {
	StepID                 int64
	Status                 string
	Description            string
	ImageURLs              []string
	ConfirmPaymentTrigger  bool
	HeaderPaymentConfirmed bool
}

func NewProductionService(repo *repository.ProductionRepository) *ProductionService {
	return &ProductionService{repo: repo}
}

func (s *ProductionService) ListSteps() ([]domain.ProductionStepTemplate, error) {
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
	status := normalizeProductionOrderStatus(order.OrderStatus)
	if isLockedProductionReadStatus(status) {
		lockReason := deriveProductionLockReason(status)
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
	}, nil
}

func (s *ProductionService) Upsert(orderID, userID int64, input ProductionWriteInput) (*domain.ProductionUpdateResult, error) {
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	input.Description = strings.TrimSpace(input.Description)
	if len(input.Description) > 500 {
		return nil, &ProductionRuleError{Err: ErrProductionDescriptionTooLong}
	}
	if input.Status != "IP" && input.Status != "CD" {
		return nil, &ProductionRuleError{Err: ErrProductionInvalidStateTransition}
	}
	if err := validateImageURLs(input.ImageURLs); err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(context.Background())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	role, err := s.repo.GetUserRole(userID)
	if err != nil {
		return nil, err
	}
	role = normalizeUserRole(role)
	if !s.repo.IsFactoryRole(role) {
		return nil, &ProductionRuleError{Err: ErrProductionNotOrderFactory}
	}

	order, err := s.repo.GetOrderForUpdateTx(tx, orderID)
	if err != nil {
		return nil, err
	}
	if order.FactoryID != userID {
		return nil, &ProductionRuleError{Err: ErrProductionNotOrderFactory}
	}
	if isLockedOrderStatus(order.OrderStatus) {
		return nil, &ProductionRuleError{Err: ErrProductionOrderLocked}
	}

	steps, err := s.repo.ListActiveStepsTx(tx)
	if err != nil {
		return nil, err
	}
	step := s.repo.StepByID(steps, input.StepID)
	if step == nil || !step.IsActive {
		return nil, &ProductionRuleError{Err: ErrProductionInvalidStep}
	}

	persisted, err := s.repo.ListByOrderIDTx(tx, orderID)
	if err != nil {
		return nil, err
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

	if step.SortOrder > 1 {
		prevStep := s.repo.StepBySortOrder(steps, step.SortOrder-1)
		prev := s.repo.GetUpdateByOrderAndStep(orderID, prevStep.StepID, inflated)
		if input.Status == "IP" && (prev == nil || prev.Status != "CD") {
			return nil, &ProductionRuleError{
				Err:     ErrProductionStepOrderViolation,
				Details: map[string]interface{}{"required_previous_step": prevStep.StepCode},
			}
		}
	}

	if current.Status == "CD" && input.Status == "IP" {
		return nil, &ProductionRuleError{Err: ErrProductionInvalidStateTransition}
	}
	if current.Status == "PD" && input.Status == "CD" {
		return nil, &ProductionRuleError{Err: ErrProductionInvalidStateTransition}
	}
	if current.Status == "RJ" && input.Status == "CD" {
		return nil, &ProductionRuleError{Err: ErrProductionInvalidStateTransition}
	}

	if active := s.repo.GetActiveInProgressStep(inflated, input.StepID); input.Status == "IP" && active != nil && active.Status == "IP" && active.StepID != input.StepID {
		return nil, &ProductionRuleError{
			Err:     ErrProductionAnotherStepInProgress,
			Details: map[string]interface{}{"in_progress_step_id": active.StepID},
		}
	}

	if input.Status == "CD" {
		if len(input.ImageURLs) < int(step.MinPhotos) {
			return nil, &ProductionRuleError{
				Err:     ErrProductionInsufficientEvidence,
				Details: map[string]interface{}{"required": step.MinPhotos, "provided": len(input.ImageURLs)},
			}
		}
		if step.IsPaymentTrigger && (!input.ConfirmPaymentTrigger || !input.HeaderPaymentConfirmed) {
			return nil, &ProductionRuleError{Err: ErrProductionPaymentConfirmRequired}
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
	if current.Status == input.Status && current.Description == input.Description && equalStringArrays(current.ImageURLs, update.ImageURLs) {
		update.UpdateID = current.UpdateID
		update.CreatedAt = current.CreatedAt
		update.LastUpdatedAt = current.LastUpdatedAt
		update.CompletedAt = current.CompletedAt
		update.RejectedReason = current.RejectedReason
		return &domain.ProductionUpdateResult{
			Update:      *current,
			OrderStatus: order.OrderStatus,
		}, nil
	}

	if err := s.repo.UpsertTx(tx, update); err != nil {
		return nil, err
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
				return nil, err
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
				return nil, err
			}
			autoProgressed = &domain.ProductionUpdateAutoProgressed{StepID: nextStep.StepID, Status: "IP"}
		}
	}

	newOrderStatus := order.OrderStatus
	switch step.StepCode {
	case "QUALITY_CONTROL":
		if input.Status == "CD" {
			newOrderStatus = "QC"
		}
	case "READY_TO_SHIP":
		if input.Status == "CD" {
			newOrderStatus = "SH"
		}
	case "SHIPPED":
		if input.Status == "CD" {
			newOrderStatus = "CP"
		}
	default:
		if input.Status == "IP" && isPreProductionOrderStatus(order.OrderStatus) {
			newOrderStatus = "PR"
		}
	}
	if input.Status == "CD" && step.StepCode == "SHIPPED" {
		newOrderStatus = "CP"
	}
	if newOrderStatus != order.OrderStatus {
		if _, err := tx.Exec(`UPDATE orders SET status = $1, updated_at = NOW() WHERE order_id = $2`, newOrderStatus, orderID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &domain.ProductionUpdateResult{
		Update:         *update,
		OrderStatus:    newOrderStatus,
		AutoProgressed: autoProgressed,
	}, nil
}

func (s *ProductionService) Reject(updateID, userID int64, reason string) (*domain.ProductionUpdate, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 1000 {
		return nil, &ProductionRuleError{Err: ErrProductionReasonRequired}
	}

	tx, err := s.repo.BeginTx(context.Background())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	role, err := s.repo.GetUserRole(userID)
	if err != nil {
		return nil, err
	}
	role = normalizeUserRole(role)

	updateCtx, err := s.repo.GetUpdateByIDForUpdateTx(tx, updateID)
	if err != nil {
		return nil, err
	}
	order, err := s.repo.GetOrderForUpdateTx(tx, updateCtx.OrderID)
	if err != nil {
		return nil, err
	}
	if isLockedOrderStatus(order.OrderStatus) {
		return nil, &ProductionRuleError{Err: ErrProductionOrderLocked}
	}

	if s.repo.IsCustomerRole(role) {
		if order.UserID != userID {
			return nil, &ProductionRuleError{Err: ErrProductionNotOrderCustomer}
		}
	} else if !s.repo.IsAdminRole(role) {
		return nil, &ProductionRuleError{Err: ErrProductionNotOrderCustomer}
	}

	if updateCtx.Status != "CD" {
		return nil, &ProductionRuleError{Err: ErrProductionInvalidStateTransition}
	}
	persisted, err := s.repo.ListByOrderIDTx(tx, updateCtx.OrderID)
	if err != nil {
		return nil, err
	}
	if s.repo.HasDownstreamInFlight(persisted, updateCtx.SortOrder) {
		return nil, &ProductionRuleError{Err: ErrProductionDownstreamInFlight}
	}
	item, err := s.repo.RejectTx(tx, updateID, reason, userID)
	if err != nil {
		return nil, err
	}
	item.StepCode = updateCtx.StepCode
	item.StepNameTH = updateCtx.StepNameTH
	item.StepNameEN = updateCtx.StepNameEN
	item.SortOrder = updateCtx.SortOrder
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return item, nil
}

func validateImageURLs(items []string) error {
	if len(items) > 6 {
		return &ProductionRuleError{Err: ErrProductionInvalidImageURL}
	}
	seen := make(map[string]struct{}, len(items))
	for _, raw := range items {
		v := strings.TrimSpace(raw)
		if v == "" || len(v) > 2048 {
			return &ProductionRuleError{Err: ErrProductionInvalidImageURL}
		}
		u, err := url.Parse(v)
		if err != nil || strings.ToLower(u.Scheme) != "https" || u.Host == "" {
			return &ProductionRuleError{Err: ErrProductionInvalidImageURL}
		}
		if _, ok := seen[v]; ok {
			return &ProductionRuleError{Err: ErrProductionInvalidImageURL}
		}
		seen[v] = struct{}{}
	}
	return nil
}

func normalizeUserRole(role string) string {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "CT":
		return "CU"
	default:
		return strings.ToUpper(strings.TrimSpace(role))
	}
}

func isLockedOrderStatus(status string) bool {
	switch normalizeProductionOrderStatus(status) {
	case "PP", "PE", "CN", "CP":
		return true
	default:
		return false
	}
}

func isLockedProductionReadStatus(status string) bool {
	switch normalizeProductionOrderStatus(status) {
	case "PP", "PE", "CN":
		return true
	default:
		return false
	}
}

func isPreProductionOrderStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "CF", "PE", "PP":
		return true
	default:
		return false
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

func normalizeProductionOrderStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "CC":
		return "CN"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func deriveProductionLockReason(status string) string {
	switch normalizeProductionOrderStatus(status) {
	case "PP":
		return "PENDING_DEPOSIT"
	case "PE":
		return "DEPOSIT_EXPIRED"
	case "CN":
		return "ORDER_CANCELLED"
	default:
		return "UNKNOWN"
	}
}

func buildProductionLockContext(order *repository.ProductionOrderContext, reason string) map[string]interface{} {
	depositAmount := roundCurrency(orderDepositAmountFallback(order))
	depositPercent := percentOf(depositAmount, order.TotalAmount)
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
		graceEnds := expiredAt.AddDate(0, 0, 3)
		return map[string]interface{}{
			"deposit_amount":      depositAmount,
			"deposit_currency":    "THB",
			"expired_at":          expiredAt,
			"grace_period_ends":   graceEnds,
			"payment_url":         fmt.Sprintf("/orders/%d/payment?stage=deposit", order.OrderID),
			"contact_factory_url": fmt.Sprintf("/chat?factory_id=%d&order_id=%d", order.FactoryID, order.OrderID),
		}
	case "ORDER_CANCELLED":
		cancelledAt := order.CreatedAt.In(thailandLocation)
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

func depositDueDateForProduction(order *repository.ProductionOrderContext) time.Time {
	due := order.CreatedAt.In(thailandLocation).AddDate(0, 0, 3)
	return time.Date(due.Year(), due.Month(), due.Day(), 23, 59, 59, 0, thailandLocation)
}

func orderDepositAmountFallback(order *repository.ProductionOrderContext) float64 {
	if order.DepositAmount > 0 {
		return order.DepositAmount
	}
	return roundCurrency(order.TotalAmount * 0.3)
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
