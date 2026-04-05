package service

import (
	"errors"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var ErrQuotationNotAccepted = errors.New("quotation must be accepted before creating order")

type OrderService struct {
	repo *repository.OrderRepository
}

func NewOrderService(repo *repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateFromQuotation(quotationID, userID int64) (*domain.Order, error) {
	src, err := s.repo.GetOrderSourceByQuotationID(quotationID, userID)
	if err != nil {
		return nil, err
	}
	if src.Status != "AC" {
		return nil, ErrQuotationNotAccepted
	}

	total := (src.PricePerPiece * float64(src.Quantity)) + src.MoldCost
	deposit := total * 0.5

	now := time.Now()
	est := now.AddDate(0, 0, int(src.LeadTimeDays))
	deliveryDate := time.Date(est.Year(), est.Month(), est.Day(), 0, 0, 0, 0, est.Location())
	order := &domain.Order{
		QuotationID:       src.QuotationID,
		UserID:            src.UserID,
		FactoryID:         src.FactoryID,
		TotalAmount:       total,
		DepositAmount:     deposit,
		Status:            "PR",
		EstimatedDelivery: &deliveryDate,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.repo.Create(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *OrderService) ListByUserID(userID int64, status string) ([]domain.Order, error) {
	return s.repo.ListByUserID(userID, strings.TrimSpace(strings.ToUpper(status)))
}

func (s *OrderService) GetByID(orderID, userID int64) (*domain.Order, error) {
	return s.repo.GetByID(orderID, userID)
}

func (s *OrderService) UpdateStatus(orderID int64, status string) error {
	return s.repo.UpdateStatus(orderID, strings.TrimSpace(strings.ToUpper(status)))
}
