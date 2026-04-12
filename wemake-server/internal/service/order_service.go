package service

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var ErrQuotationNotAccepted = errors.New("quotation must be accepted before creating order")
var ErrShipOrderInvalid = errors.New("tracking_no and courier are required")

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
	uid := userID
	_ = s.repo.InsertActivity(order.OrderID, &uid, "ORDER_CREATED", map[string]interface{}{
		"status": order.Status, "quote_id": order.QuotationID,
	})
	return order, nil
}

func (s *OrderService) List(userID int64, role string, status string) ([]domain.Order, error) {
	st := strings.TrimSpace(strings.ToUpper(status))
	if role == domain.RoleFactory {
		return s.repo.ListByFactoryID(userID, st)
	}
	return s.repo.ListByUserID(userID, st)
}

func (s *OrderService) GetByID(orderID, userID int64, role string) (*domain.Order, error) {
	return s.repo.GetByParticipant(orderID, userID, role)
}

func (s *OrderService) UpdateStatus(orderID int64, status string, actorUserID *int64) error {
	if err := s.repo.UpdateStatus(orderID, strings.TrimSpace(strings.ToUpper(status))); err != nil {
		return err
	}
	return s.repo.InsertActivity(orderID, actorUserID, "ORDER_STATUS", map[string]interface{}{
		"status": strings.TrimSpace(strings.ToUpper(status)),
	})
}

func (s *OrderService) ListActivity(orderID int64) ([]domain.OrderActivityEntry, error) {
	return s.repo.ListActivity(orderID)
}

func (s *OrderService) MarkShipped(orderID, factoryID int64, trackingNo, courier string) error {
	trackingNo = strings.TrimSpace(trackingNo)
	courier = strings.TrimSpace(courier)
	if trackingNo == "" || courier == "" {
		return ErrShipOrderInvalid
	}
	order, err := s.repo.GetByParticipant(orderID, factoryID, domain.RoleFactory)
	if err != nil {
		return err
	}
	if order.Status != "PR" && order.Status != "QC" && order.Status != "SH" {
		return sql.ErrNoRows
	}
	if err := s.repo.MarkShipped(orderID, factoryID, trackingNo, courier); err != nil {
		return err
	}
	uid := factoryID
	return s.repo.InsertActivity(orderID, &uid, "ORDER_SHIPPED", map[string]interface{}{
		"status":      "SH",
		"tracking_no": trackingNo,
		"courier":     courier,
	})
}
