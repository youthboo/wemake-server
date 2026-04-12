package service

import (
	"errors"
	"strings"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var ErrInvalidDisputeStatus = errors.New("status must be RS or CL")

type DisputeService struct {
	repo *repository.DisputeRepository
}

func NewDisputeService(repo *repository.DisputeRepository) *DisputeService {
	return &DisputeService{repo: repo}
}

func (s *DisputeService) Create(orderID, openedBy int64, reason string) (*domain.Dispute, error) {
	d := &domain.Dispute{
		OrderID:  orderID,
		OpenedBy: openedBy,
		Reason:   reason,
	}
	if err := s.repo.Create(d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *DisputeService) GetByOrderID(orderID int64) (*domain.Dispute, error) {
	return s.repo.GetByOrderID(orderID)
}

func (s *DisputeService) UpdateStatus(disputeID int64, status string, resolution *string) error {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "RS" && status != "CL" {
		return ErrInvalidDisputeStatus
	}
	return s.repo.UpdateStatus(disputeID, status, resolution)
}
