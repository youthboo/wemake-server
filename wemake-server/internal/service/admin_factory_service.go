package service

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var (
	ErrFactoryApprovalState = errors.New("invalid factory approval state transition")
	ErrReasonRequired       = errors.New("reason must be at least 10 characters")
)

type AdminFactoryService struct {
	factories  *repository.FactoryRepository
	audit      *repository.AdminAuditRepository
	commission *repository.CommissionRepository
}

func NewAdminFactoryService(factories *repository.FactoryRepository, audit *repository.AdminAuditRepository, commission *repository.CommissionRepository) *AdminFactoryService {
	return &AdminFactoryService{factories: factories, audit: audit, commission: commission}
}

func (s *AdminFactoryService) Approve(factoryID, actorID int64, note string, ip *string) error {
	detail, err := s.factories.GetAdminDetail(factoryID)
	if err != nil {
		return err
	}
	if detail.ApprovalStatus != "PE" && detail.ApprovalStatus != "RJ" {
		return ErrFactoryApprovalState
	}
	if err := s.factories.UpdateApprovalStatus(factoryID, "AP", &actorID, nil, true); err != nil {
		return err
	}
	return s.insertAudit(actorID, "FACTORY_APPROVE", "factory", factoryID, map[string]interface{}{"before": detail, "note": note}, ip)
}

func (s *AdminFactoryService) Reject(factoryID, actorID int64, reason string, ip *string) error {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) < 10 {
		return ErrReasonRequired
	}
	detail, err := s.factories.GetAdminDetail(factoryID)
	if err != nil {
		return err
	}
	if detail.ApprovalStatus != "PE" && detail.ApprovalStatus != "AP" {
		return ErrFactoryApprovalState
	}
	if err := s.factories.UpdateApprovalStatus(factoryID, "RJ", &actorID, &reason, false); err != nil {
		return err
	}
	return s.insertAudit(actorID, "FACTORY_REJECT", "factory", factoryID, map[string]interface{}{"before": detail, "reason": reason}, ip)
}

func (s *AdminFactoryService) Suspend(factoryID, actorID int64, reason string, ip *string) error {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) < 10 {
		return ErrReasonRequired
	}
	detail, err := s.factories.GetAdminDetail(factoryID)
	if err != nil {
		return err
	}
	if detail.ApprovalStatus != "AP" {
		return ErrFactoryApprovalState
	}
	if err := s.factories.UpdateApprovalStatus(factoryID, "SU", &actorID, &reason, false); err != nil {
		return err
	}
	return s.insertAudit(actorID, "FACTORY_SUSPEND", "factory", factoryID, map[string]interface{}{"before": detail, "reason": reason}, ip)
}

func (s *AdminFactoryService) Unsuspend(factoryID, actorID int64, note string, ip *string) error {
	detail, err := s.factories.GetAdminDetail(factoryID)
	if err != nil {
		return err
	}
	if detail.ApprovalStatus != "SU" {
		return ErrFactoryApprovalState
	}
	if err := s.factories.UpdateApprovalStatus(factoryID, "AP", &actorID, nil, true); err != nil {
		return err
	}
	return s.insertAudit(actorID, "FACTORY_UNSUSPEND", "factory", factoryID, map[string]interface{}{"before": detail, "note": note}, ip)
}

func (s *AdminFactoryService) ToggleVerification(factoryID, actorID int64, isVerified bool, note string, ip *string) error {
	status := "RJ"
	if isVerified {
		status = "AP"
	}
	var reason *string
	if !isVerified && strings.TrimSpace(note) != "" {
		n := strings.TrimSpace(note)
		reason = &n
	}
	if err := s.factories.UpdateApprovalStatus(factoryID, status, &actorID, reason, isVerified); err != nil {
		return err
	}
	return s.insertAudit(actorID, "FACTORY_VERIFICATION_PATCH", "factory", factoryID, map[string]interface{}{"is_verified": isVerified, "note": note}, ip)
}

func (s *AdminFactoryService) HydrateAdminDetail(factoryID int64) (*domain.AdminFactoryDetail, error) {
	item, err := s.factories.GetAdminDetail(factoryID)
	if err != nil {
		return nil, err
	}
	item.CommissionOverride, _ = s.commission.GetActiveRuleForFactory(factoryID)
	item.IsCommissionExempt, _ = s.commission.FactoryHasActiveExemption(factoryID)
	return item, nil
}

func (s *AdminFactoryService) insertAudit(actorID int64, action, targetType string, targetID int64, payload interface{}, ip *string) error {
	raw, _ := json.Marshal(payload)
	return s.audit.Insert(&domain.AdminAuditLog{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   strconv.FormatInt(targetID, 10),
		Payload:    raw,
		IPAddress:  ip,
		CreatedAt:  time.Now().UTC(),
	})
}
