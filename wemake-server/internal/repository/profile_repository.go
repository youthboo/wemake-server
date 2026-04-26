package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type ProfileRepository struct {
	db *sqlx.DB
}

func NewProfileRepository(db *sqlx.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) GetProfile(userID int64) (*domain.ProfileResponse, error) {
	var user domain.User
	if err := r.db.Get(&user, `
		SELECT user_id, role, email, phone, avatar_url, bio, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE user_id = $1
	`, userID); err != nil {
		return nil, err
	}
	out := &domain.ProfileResponse{
		UserID:    user.UserID,
		Role:      user.Role,
		Email:     user.Email,
		Phone:     user.Phone,
		AvatarURL: user.AvatarURL,
		Bio:       user.Bio,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	switch user.Role {
	case domain.RoleCustomer:
		var customer domain.CustomerProfile
		if err := r.db.Get(&customer, `
			SELECT user_id, first_name, last_name, address_line1, sub_district, district, province, postal_code
			FROM customers
			WHERE user_id = $1
		`, userID); err != nil {
			return nil, err
		}
		out.FirstName = &customer.FirstName
		out.LastName = &customer.LastName
		out.Address = &domain.ProfileAddress{
			Line1: customer.AddressLine1, SubDistrict: customer.SubDistrict, District: customer.District,
			Province: customer.Province, PostalCode: customer.PostalCode,
		}
	case domain.RoleFactory:
		var factory domain.FactoryProfile
		if err := r.db.Get(&factory, `
			SELECT user_id, factory_name, factory_type_id, tax_id, province_id, specialization, min_order, lead_time_desc,
			       is_verified, verified_at, description, price_range
			FROM factory_profiles
			WHERE user_id = $1
		`, userID); err != nil {
			return nil, err
		}
		out.FactoryName = &factory.FactoryName
		out.FactoryTypeID = &factory.FactoryTypeID
		out.TaxID = stringPtrIfNotEmpty(factory.TaxID)
		out.ProvinceID = factory.ProvinceID
		out.Specialization = factory.Specialization
		out.MinOrder = factory.MinOrder
		out.LeadTimeDesc = factory.LeadTimeDesc
		out.IsVerified = &factory.IsVerified
		out.VerifiedAt = factory.VerifiedAt
		out.Description = factory.Description
		out.PriceRange = factory.PriceRange
	}
	return out, nil
}

func (r *ProfileRepository) UpdateCustomerProfile(userID int64, user *domain.User, customer *domain.CustomerProfile) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		UPDATE users SET phone = $1, bio = $2, updated_at = NOW()
		WHERE user_id = $3
	`, user.Phone, nullableStringPtr(user.Bio), userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE customers
		SET first_name = $1, last_name = $2, address_line1 = $3, sub_district = $4, district = $5, province = $6, postal_code = $7
		WHERE user_id = $8
	`, customer.FirstName, customer.LastName, nullableStringPtr(customer.AddressLine1), nullableStringPtr(customer.SubDistrict), nullableStringPtr(customer.District), nullableStringPtr(customer.Province), nullableStringPtr(customer.PostalCode), userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ProfileRepository) UpdateFactoryProfile(userID int64, user *domain.User, factory *domain.FactoryProfile) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		UPDATE users SET phone = $1, bio = $2, updated_at = NOW()
		WHERE user_id = $3
	`, user.Phone, nullableStringPtr(user.Bio), userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE factory_profiles
		SET description = $1, specialization = $2, min_order = $3, lead_time_desc = $4, price_range = $5
		WHERE user_id = $6
	`, nullableStringPtr(factory.Description), nullableStringPtr(factory.Specialization), nullableInt64Value(factory.MinOrder), nullableStringPtr(factory.LeadTimeDesc), nullableStringPtr(factory.PriceRange), userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ProfileRepository) UpdateAvatar(userID int64, avatarURL string) error {
	_, err := r.db.Exec(`UPDATE users SET avatar_url = $1, updated_at = NOW() WHERE user_id = $2`, avatarURL, userID)
	return err
}

func (r *ProfileRepository) GetSummary(userID int64, role string) (*domain.ProfileSummary, error) {
	out := &domain.ProfileSummary{}
	switch role {
	case domain.RoleFactory:
		err := r.db.Get(out, `
			SELECT
				COUNT(*)::bigint AS total_orders,
				COUNT(*) FILTER (WHERE o.status IN ('PP','PR','WF','QC','SH','DL','AC'))::bigint AS pending_orders,
				COUNT(*) FILTER (WHERE o.status = 'CP')::bigint AS completed_orders,
				COUNT(*) FILTER (WHERE o.status = 'CC')::bigint AS cancelled_orders,
				COALESCE(SUM(o.total_amount) FILTER (WHERE o.status = 'CP'), 0)::float8 AS total_spend_thb,
				0::bigint AS total_rfqs,
				0::bigint AS active_rfqs,
				0::bigint AS reviews_given,
				fp.rating::float8 AS average_rating_received
			FROM orders o
			RIGHT JOIN factory_profiles fp ON fp.user_id = $1
			WHERE o.factory_id = $1 OR o.factory_id IS NULL
			GROUP BY fp.rating
		`, userID)
		return out, err
	default:
		err := r.db.Get(out, `
			SELECT
				COUNT(*)::bigint AS total_orders,
				COUNT(*) FILTER (WHERE o.status IN ('PP','PR','WF','QC','SH','DL','AC'))::bigint AS pending_orders,
				COUNT(*) FILTER (WHERE o.status = 'CP')::bigint AS completed_orders,
				COUNT(*) FILTER (WHERE o.status = 'CC')::bigint AS cancelled_orders,
				COALESCE(SUM(o.total_amount) FILTER (WHERE o.status = 'CP'), 0)::float8 AS total_spend_thb,
				(SELECT COUNT(*)::bigint FROM rfqs WHERE user_id = $1) AS total_rfqs,
				(SELECT COUNT(*)::bigint FROM rfqs WHERE user_id = $1 AND status NOT IN ('CC','CL')) AS active_rfqs,
				(SELECT COUNT(*)::bigint FROM factory_reviews WHERE user_id = $1 AND deleted_at IS NULL) AS reviews_given
			FROM orders o
			WHERE o.user_id = $1
		`, userID)
		return out, err
	}
}

func (r *ProfileRepository) ListTransactions(userID int64, page, limit int, txType, status string) ([]domain.TransactionListItem, int64, float64, float64, error) {
	offset := (page - 1) * limit
	where := []string{"w.user_id = $1"}
	args := []interface{}{userID}
	argPos := 2
	if txType != "" && strings.ToLower(txType) != "all" {
		where = append(where, fmt.Sprintf("t.type = $%d", argPos))
		args = append(args, strings.ToUpper(strings.TrimSpace(txType)))
		argPos++
	}
	if status != "" {
		where = append(where, fmt.Sprintf("t.status = $%d", argPos))
		args = append(args, strings.ToUpper(strings.TrimSpace(status)))
		argPos++
	}
	cond := strings.Join(where, " AND ")
	var total int64
	if err := r.db.Get(&total, `SELECT COUNT(*) FROM transactions t INNER JOIN wallets w ON w.wallet_id = t.wallet_id WHERE `+cond, args...); err != nil {
		return nil, 0, 0, 0, err
	}
	query := `SELECT t.tx_id, t.type, t.amount::float8 AS amount, t.status, t.created_at, t.order_id
		FROM transactions t
		INNER JOIN wallets w ON w.wallet_id = t.wallet_id
		WHERE ` + cond + fmt.Sprintf(` ORDER BY t.created_at DESC LIMIT $%d OFFSET $%d`, argPos, argPos+1)
	args = append(args, limit, offset)
	type row struct {
		TxID      string    `db:"tx_id"`
		Type      string    `db:"type"`
		Amount    float64   `db:"amount"`
		Status    string    `db:"status"`
		CreatedAt time.Time `db:"created_at"`
		OrderID   *int64    `db:"order_id"`
	}
	var rows []row
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, 0, 0, 0, err
	}
	items := make([]domain.TransactionListItem, 0, len(rows))
	var totalIn, totalOut float64
	for _, row := range rows {
		item := mapProfileTransaction(row.TxID, row.Type, row.Amount, row.Status, row.OrderID, row.CreatedAt)
		items = append(items, item)
		if item.Direction == "in" {
			totalIn += item.Amount
		} else {
			totalOut += item.Amount
		}
	}
	return items, total, totalIn, totalOut, nil
}

func (r *ProfileRepository) ListMyReviews(userID int64, page, limit int) ([]domain.UserReviewListItem, int64, error) {
	return r.listReviews(`
		FROM factory_reviews fr
		JOIN factory_profiles fp ON fp.user_id = fr.factory_id
		WHERE fr.user_id = $1 AND fr.deleted_at IS NULL
	`, userID, page, limit, false)
}

func (r *ProfileRepository) ListReceivedReviews(factoryID int64, page, limit int) ([]domain.UserReviewListItem, int64, error) {
	return r.listReviews(`
		FROM factory_reviews fr
		JOIN factory_profiles fp ON fp.user_id = fr.factory_id
		LEFT JOIN customers c ON c.user_id = fr.user_id
		WHERE fr.factory_id = $1 AND fr.deleted_at IS NULL
	`, factoryID, page, limit, true)
}

func (r *ProfileRepository) listReviews(fromWhere string, userID int64, page, limit int, includeReviewer bool) ([]domain.UserReviewListItem, int64, error) {
	offset := (page - 1) * limit
	var total int64
	if err := r.db.Get(&total, `SELECT COUNT(*) `+fromWhere, userID); err != nil {
		return nil, 0, err
	}
	selectReviewer := `NULL::text AS reviewer_name`
	if includeReviewer {
		selectReviewer = `NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), '') AS reviewer_name`
	}
	query := `SELECT
		fr.review_id, fr.order_id, fr.factory_id, fp.factory_name, fp.image_url AS factory_avatar,
		` + selectReviewer + `,
		fr.rating, COALESCE(fr.comment, '') AS comment,
		(fr.created_at > NOW() - INTERVAL '7 days') AS is_editable,
		fr.created_at, fr.updated_at
	` + fromWhere + ` ORDER BY fr.created_at DESC LIMIT $2 OFFSET $3`
	var items []domain.UserReviewListItem
	if err := r.db.Select(&items, query, userID, limit, offset); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ProfileRepository) GetNotificationPreference(userID int64) (*domain.NotificationPreference, error) {
	var item domain.NotificationPreference
	err := r.db.Get(&item, `
		SELECT user_id, order_updates, rfq_updates, chat_messages, promotions, email_enabled, push_enabled, updated_at
		FROM user_notification_preferences
		WHERE user_id = $1
	`, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &domain.NotificationPreference{
				UserID: userID, OrderUpdates: true, RFQUpdates: true, ChatMessages: true, Promotions: false, EmailEnabled: true, PushEnabled: true,
			}, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *ProfileRepository) UpsertNotificationPreference(item *domain.NotificationPreference) error {
	return r.db.QueryRow(`
		INSERT INTO user_notification_preferences (user_id, order_updates, rfq_updates, chat_messages, promotions, email_enabled, push_enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (user_id) DO UPDATE
		SET order_updates = EXCLUDED.order_updates,
		    rfq_updates = EXCLUDED.rfq_updates,
		    chat_messages = EXCLUDED.chat_messages,
		    promotions = EXCLUDED.promotions,
		    email_enabled = EXCLUDED.email_enabled,
		    push_enabled = EXCLUDED.push_enabled,
		    updated_at = NOW()
		RETURNING updated_at
	`, item.UserID, item.OrderUpdates, item.RFQUpdates, item.ChatMessages, item.Promotions, item.EmailEnabled, item.PushEnabled).Scan(&item.UpdatedAt)
}

func stringPtrIfNotEmpty(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

func mapProfileTransaction(txID, txType string, amount float64, status string, orderID *int64, createdAt time.Time) domain.TransactionListItem {
	typeLabel := map[string]string{
		"DP": "ชำระมัดจำ",
		"WD": "ถอนเงิน",
		"BU": "เติมเงิน",
		"SC": "รับชำระจากลูกค้า",
		"RF": "คืนเงิน",
	}[txType]
	if typeLabel == "" {
		typeLabel = txType
	}
	direction := "out"
	if txType == "BU" || txType == "SC" || txType == "RF" {
		direction = "in"
	}
	statusLabel := map[string]string{
		"ST": "สำเร็จ",
		"PT": "รอดำเนินการ",
		"RJ": "ไม่สำเร็จ",
	}[status]
	if statusLabel == "" {
		statusLabel = status
	}
	description := typeLabel
	var referenceType *string
	if orderID != nil {
		rt := "order"
		referenceType = &rt
		description = fmt.Sprintf("%s Order #%d", typeLabel, *orderID)
	}
	return domain.TransactionListItem{
		TxID:          txID,
		Type:          txType,
		TypeLabel:     typeLabel,
		Amount:        amount,
		Currency:      "THB",
		Direction:     direction,
		Description:   description,
		ReferenceID:   orderID,
		ReferenceType: referenceType,
		Status:        status,
		StatusLabel:   statusLabel,
		CreatedAt:     createdAt,
	}
}
