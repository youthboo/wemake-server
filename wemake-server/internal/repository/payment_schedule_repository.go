package repository

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type PaymentScheduleRepository struct {
	db *sqlx.DB
}

func NewPaymentScheduleRepository(db *sqlx.DB) *PaymentScheduleRepository {
	return &PaymentScheduleRepository{db: db}
}

func (r *PaymentScheduleRepository) ListByOrderID(orderID int64) ([]domain.PaymentSchedule, error) {
	var items []domain.PaymentSchedule
	err := r.db.Select(&items, `
		SELECT schedule_id, order_id, installment_no, due_date, amount, status, paid_at, created_at
		FROM payment_schedules
		WHERE order_id = $1
		ORDER BY installment_no ASC
	`, orderID)
	return items, err
}

func (r *PaymentScheduleRepository) Create(s *domain.PaymentSchedule) error {
	return r.db.QueryRow(`
		INSERT INTO payment_schedules (order_id, installment_no, due_date, amount, status)
		VALUES ($1, $2, $3, $4, 'PE')
		RETURNING schedule_id, created_at
	`, s.OrderID, s.InstallmentNo, s.DueDate, s.Amount).
		Scan(&s.ScheduleID, &s.CreatedAt)
}

func (r *PaymentScheduleRepository) CreateTx(tx *sqlx.Tx, s *domain.PaymentSchedule) error {
	return tx.QueryRow(`
		INSERT INTO payment_schedules (order_id, installment_no, due_date, amount, status)
		VALUES ($1, $2, $3, $4, 'PE')
		RETURNING schedule_id, created_at
	`, s.OrderID, s.InstallmentNo, s.DueDate, s.Amount).
		Scan(&s.ScheduleID, &s.CreatedAt)
}

func (r *PaymentScheduleRepository) PatchStatus(scheduleID int64, status string) error {
	var paidAt interface{}
	if status == "PD" {
		now := time.Now()
		paidAt = now
	}
	res, err := r.db.Exec(`
		UPDATE payment_schedules
		SET status = $1,
		    paid_at = CASE WHEN $1 = 'PD' THEN $2 ELSE paid_at END
		WHERE schedule_id = $3
	`, status, paidAt, scheduleID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PaymentScheduleRepository) PatchStatusByOrderAndInstallmentTx(tx *sqlx.Tx, orderID int64, installmentNo int, status string) error {
	var paidAt interface{}
	if status == "PD" {
		now := time.Now()
		paidAt = now
	}
	res, err := tx.Exec(`
		UPDATE payment_schedules
		SET status = $1,
		    paid_at = CASE WHEN $1 = 'PD' THEN $2 ELSE paid_at END
		WHERE order_id = $3 AND installment_no = $4
	`, status, paidAt, orderID, installmentNo)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
