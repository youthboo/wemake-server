package wallet

import (
	"errors"
	"strings"

	"github.com/yourusername/wemake/internal/domain"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
)

var (
	ErrInvalidDisputeCategory = errors.New("category must be NR, ND, or OT")
	ErrRefundSlipRequired     = errors.New("refund slip is required")
	ErrInvalidResolution      = errors.New("action must be 'refund' or 'reject'")
)

type DisputeService struct {
	repo *walletrepo.DisputeRepository
}

func NewDisputeService(repo *walletrepo.DisputeRepository) *DisputeService {
	return &DisputeService{repo: repo}
}

// DisputeContact carries the refund destination + callback details.
type DisputeContact struct {
	RefundAccount     string
	RefundAccountName string
	ContactEmail      string
	ContactPhone      string
}

func (s *DisputeService) Create(orderID, openedBy int64, category, reason string, evidence []string, contact DisputeContact) (*domain.Dispute, error) {
	category = strings.ToUpper(strings.TrimSpace(category))
	if category != domain.DisputeCategoryNotReceived &&
		category != domain.DisputeCategoryNotAsDesc &&
		category != domain.DisputeCategoryOther {
		return nil, ErrInvalidDisputeCategory
	}
	ev := domain.StringArray{}
	for _, u := range evidence {
		if t := strings.TrimSpace(u); t != "" {
			ev = append(ev, t)
		}
		if len(ev) >= 5 {
			break
		}
	}
	strPtr := func(v string) *string {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		return &v
	}
	d := &domain.Dispute{
		OrderID:           orderID,
		OpenedBy:          openedBy,
		Category:          category,
		Reason:            strings.TrimSpace(reason),
		EvidenceURLs:      ev,
		RefundAccount:     strPtr(contact.RefundAccount),
		RefundAccountName: strPtr(contact.RefundAccountName),
		ContactEmail:      strPtr(contact.ContactEmail),
		ContactPhone:      strPtr(contact.ContactPhone),
	}
	if err := s.repo.Create(d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *DisputeService) GetByOrderID(orderID int64) (*domain.Dispute, error) {
	return s.repo.GetByOrderID(orderID)
}

func (s *DisputeService) GetByID(disputeID int64) (*domain.Dispute, error) {
	return s.repo.GetByID(disputeID)
}

// RequestReturn (superadmin): approve complaint, ask customer to ship goods back.
func (s *DisputeService) RequestReturn(disputeID, resolvedBy int64, note string) error {
	return s.repo.RequestReturn(disputeID, resolvedBy, strings.TrimSpace(note))
}

// SubmitReturn (customer): attach return-shipping evidence.
func (s *DisputeService) SubmitReturn(disputeID, customerID int64, tracking, courier, note string, evidence []string) error {
	ev := domain.StringArray{}
	for _, u := range evidence {
		if t := strings.TrimSpace(u); t != "" {
			ev = append(ev, t)
		}
		if len(ev) >= 5 {
			break
		}
	}
	return s.repo.SubmitReturn(disputeID, customerID, strings.TrimSpace(tracking), strings.TrimSpace(courier), strings.TrimSpace(note), ev)
}

// Reject closes the ticket without a refund.
func (s *DisputeService) Reject(disputeID, resolvedBy int64, resolution string) error {
	return s.repo.Reject(disputeID, resolvedBy, strings.TrimSpace(resolution))
}

// Refund refunds the customer (amount<=0 = full); slipURL is the transfer slip.
func (s *DisputeService) Refund(disputeID, resolvedBy int64, amount float64, resolution, slipURL string) (float64, error) {
	if strings.TrimSpace(slipURL) == "" {
		return 0, ErrRefundSlipRequired
	}
	return s.repo.Refund(disputeID, resolvedBy, amount, strings.TrimSpace(resolution), strings.TrimSpace(slipURL))
}
