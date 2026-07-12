package admin

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type AdminWithdrawalRepository struct {
	db *sqlx.DB
}

func NewAdminWithdrawalRepository(db *sqlx.DB) *AdminWithdrawalRepository {
	return &AdminWithdrawalRepository{db: db}
}

func (r *AdminWithdrawalRepository) ListAdmin(status string, factoryID *int64, page, pageSize int) ([]domain.AdminWithdrawalListItem, int, error) {
	page, pageSize = normalizePage(page, pageSize)
	where := "1=1"
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND wr.status = $%d", len(args))
	}
	if factoryID != nil {
		args = append(args, *factoryID)
		where += fmt.Sprintf(" AND wr.factory_id = $%d", len(args))
	}
	var total int
	if err := r.db.Get(&total, `SELECT COUNT(*) FROM withdrawal_requests wr WHERE `+where, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	items := []domain.AdminWithdrawalListItem{}
	if err := r.db.Select(&items, `
		SELECT wr.request_id, wr.factory_id, COALESCE(fp.factory_name, 'Factory #' || wr.factory_id::text) AS factory_name,
		       wr.amount, wr.bank_account_no, wr.bank_name, wr.account_name,
		       wr.status, wr.processed_at, wr.note, wr.slip_url, wr.created_at
		FROM withdrawal_requests wr
		LEFT JOIN factory_profiles fp ON fp.user_id = wr.factory_id
		WHERE `+where+`
		ORDER BY created_at DESC, request_id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
