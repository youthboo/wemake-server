package wallet

import (
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

// Order statuses on which a customer may open a complaint/refund ticket:
// paid & in progress or delivered, but not completed/cancelled/already disputed.
// 'CP' (completed) is handled separately — see disputeWindow below (allowed
// only within a grace period and only if the customer hasn't reviewed yet).
var disputableOrderStatuses = map[string]bool{
	"PD": true, // payment done
	"PR": true, // production
	"WF": true, // waiting final payment
	"QC": true, // quality check
	"SH": true, // shipping
	"DL": true, // delivered
}

// disputeWindowAfterCompletion — a customer may still complain about a
// completed order within this many days of completion (e.g. "ไม่ได้รับสินค้า"
// noticed after the auto-close). Once a review is submitted, the case is
// considered settled and can no longer be disputed.
const disputeWindowAfterCompletion = 14 * 24 * time.Hour

var (
	ErrOrderNotDisputable     = errors.New("order is not in a disputable state")
	ErrDisputeWindowExpired   = errors.New("dispute window has expired for this completed order")
	ErrAlreadyReviewed        = errors.New("order has already been reviewed")
	ErrNotOrderOwner          = errors.New("only the order customer can open a dispute")
	ErrDisputeAlreadyResolved = errors.New("dispute already resolved")
	ErrReturnNotExpected      = errors.New("dispute is not awaiting a return shipment")
	ErrReturnReasonRequired   = errors.New("a reason/instruction is required to request a return")
	ErrInvalidRefundAmount    = errors.New("refund amount must be > 0 and <= order total")
)

type DisputeRepository struct {
	db *sqlx.DB
}

func NewDisputeRepository(db *sqlx.DB) *DisputeRepository {
	return &DisputeRepository{db: db}
}

// Create opens a dispute after verifying the opener owns the order and the
// order is in a disputable state. Also flips the order to Disputed ('DP').
func (r *DisputeRepository) Create(d *domain.Dispute) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		customerID  int64
		status      string
		completedAt *time.Time
	)
	if err := tx.QueryRow(`
		SELECT customer_id, TRIM(status), completed_at FROM orders WHERE order_id = $1 FOR UPDATE
	`, d.OrderID).Scan(&customerID, &status, &completedAt); err != nil {
		return err
	}
	if customerID != d.OpenedBy {
		return ErrNotOrderOwner
	}
	if status == "CP" {
		// Completed orders get a grace period, and only if not reviewed yet —
		// a submitted review means the customer already signed off on the order.
		if completedAt == nil || time.Since(*completedAt) > disputeWindowAfterCompletion {
			return ErrDisputeWindowExpired
		}
		var reviewed bool
		if err := tx.Get(&reviewed, `SELECT EXISTS(SELECT 1 FROM factory_reviews WHERE order_id = $1)`, d.OrderID); err != nil {
			return err
		}
		if reviewed {
			return ErrAlreadyReviewed
		}
	} else if !disputableOrderStatuses[status] {
		return ErrOrderNotDisputable
	}

	evidence := d.EvidenceURLs
	if evidence == nil {
		evidence = domain.StringArray{}
	}
	if err := tx.QueryRow(`
		INSERT INTO disputes (order_id, opened_by, category, reason, evidence_urls, status,
		                      refund_account, refund_account_name, contact_email, contact_phone,
		                      prior_order_status)
		VALUES ($1, $2, $3, $4, $5, 'OP', $6, $7, $8, $9, $10)
		RETURNING dispute_id, created_at, updated_at
	`, d.OrderID, d.OpenedBy, d.Category, d.Reason, evidence,
		d.RefundAccount, d.RefundAccountName, d.ContactEmail, d.ContactPhone, status).
		Scan(&d.DisputeID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		// unique partial index → an OP dispute already exists for this order
		return err
	}

	if _, err := tx.Exec(`
		UPDATE orders SET status = 'DP', updated_at = NOW() WHERE order_id = $1
	`, d.OrderID); err != nil {
		return err
	}
	d.Status = "OP"
	return tx.Commit()
}

const disputeCols = `dispute_id, order_id, opened_by, category, reason, evidence_urls,
	refund_account, refund_account_name, contact_email, contact_phone,
	return_tracking_no, return_courier, return_note, return_evidence_urls,
	return_requested_at, return_submitted_at, prior_order_status,
	status, resolution, refund_amount, refund_slip_url, resolved_by, resolved_at, created_at, updated_at`

func (r *DisputeRepository) GetByOrderID(orderID int64) (*domain.Dispute, error) {
	var item domain.Dispute
	err := r.db.Get(&item, `SELECT `+disputeCols+`
		FROM disputes WHERE order_id = $1 ORDER BY created_at DESC LIMIT 1`, orderID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DisputeRepository) GetByID(disputeID int64) (*domain.Dispute, error) {
	var item domain.Dispute
	err := r.db.Get(&item, `SELECT `+disputeCols+` FROM disputes WHERE dispute_id = $1`, disputeID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// openStatuses = states a superadmin can still act on (not yet finalized).
func isActionable(status string) bool {
	return status == "OP" || status == "RT" || status == "RC"
}

// RequestReturn approves the complaint and asks the customer to ship the goods
// back (OP → RT). The order stays Disputed ('DP'). note (instructions to the
// customer — e.g. return address) is required so the customer knows what to do.
func (r *DisputeRepository) RequestReturn(disputeID, resolvedBy int64, note string) error {
	if len(note) < 10 {
		return ErrReturnReasonRequired
	}
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	if err := tx.QueryRow(`SELECT TRIM(status) FROM disputes WHERE dispute_id = $1 FOR UPDATE`, disputeID).Scan(&status); err != nil {
		return err
	}
	if status != "OP" {
		return ErrDisputeAlreadyResolved
	}
	_, err = tx.Exec(`
		UPDATE disputes
		SET status = 'RT', resolution = NULLIF($2,''), return_requested_at = NOW(), updated_at = NOW()
		WHERE dispute_id = $1
	`, disputeID, note)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// SubmitReturn is the customer attaching return-shipping evidence (RT → RC).
func (r *DisputeRepository) SubmitReturn(disputeID, customerID int64, tracking, courier, note string, evidence domain.StringArray) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	var openedBy int64
	if err := tx.QueryRow(`SELECT TRIM(status), opened_by FROM disputes WHERE dispute_id = $1 FOR UPDATE`, disputeID).Scan(&status, &openedBy); err != nil {
		return err
	}
	if openedBy != customerID {
		return ErrNotOrderOwner
	}
	if status != "RT" {
		return ErrReturnNotExpected
	}
	if evidence == nil {
		evidence = domain.StringArray{}
	}
	_, err = tx.Exec(`
		UPDATE disputes
		SET status = 'RC',
		    return_tracking_no = NULLIF($2,''), return_courier = NULLIF($3,''),
		    return_note = NULLIF($4,''), return_evidence_urls = $5,
		    return_submitted_at = NOW(), updated_at = NOW()
		WHERE dispute_id = $1
	`, disputeID, tracking, courier, note, evidence)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Reject closes a dispute as rejected and returns the order to whatever status
// it was in before the dispute was opened (falls back to 'PD' for disputes
// created before prior_order_status was tracked). Allowed from OP/RT/RC.
func (r *DisputeRepository) Reject(disputeID, resolvedBy int64, resolution string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var orderID int64
	var status string
	var priorStatus *string
	if err := tx.QueryRow(`
		SELECT order_id, TRIM(status), prior_order_status FROM disputes WHERE dispute_id = $1 FOR UPDATE
	`, disputeID).Scan(&orderID, &status, &priorStatus); err != nil {
		return err
	}
	if !isActionable(status) {
		return ErrDisputeAlreadyResolved
	}
	if _, err := tx.Exec(`
		UPDATE disputes
		SET status = 'RJ', resolution = $2, resolved_by = $3, resolved_at = NOW(), updated_at = NOW()
		WHERE dispute_id = $1
	`, disputeID, resolution, resolvedBy); err != nil {
		return err
	}
	restoreStatus := "PD"
	if priorStatus != nil && *priorStatus != "" {
		restoreStatus = *priorStatus
	}
	if _, err := tx.Exec(`
		UPDATE orders SET status = $2, updated_at = NOW() WHERE order_id = $1 AND TRIM(status) = 'DP'
	`, orderID, restoreStatus); err != nil {
		return err
	}
	return tx.Commit()
}

// Refund resolves a dispute by refunding the customer. amount<=0 means "full"
// (the whole order total). Allowed from OP (immediate refund, e.g. goods never
// received) or RC (after the returned goods are inspected).
//   - FULL refund  → reverse the still-held escrow legs and cancel the order (CC).
//   - PARTIAL refund → pay the refund out of the platform's escrow-in hold and
//     let the order continue (back to PD); the factory keeps its receivable for
//     the goods the customer kept.
// A refund transaction with the superadmin's transfer slip is always recorded.
func (r *DisputeRepository) Refund(disputeID, resolvedBy int64, amount float64, resolution, slipURL string) (float64, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var orderID int64
	var status string
	var priorStatus *string
	if err := tx.QueryRow(`
		SELECT order_id, TRIM(status), prior_order_status FROM disputes WHERE dispute_id = $1 FOR UPDATE
	`, disputeID).Scan(&orderID, &status, &priorStatus); err != nil {
		return 0, err
	}
	if !isActionable(status) {
		return 0, ErrDisputeAlreadyResolved
	}

	var total float64
	if err := tx.QueryRow(`
		SELECT COALESCE(total_amount, 0)::float8 FROM orders WHERE order_id = $1
	`, orderID).Scan(&total); err != nil {
		return 0, err
	}
	refundAmount := amount
	if refundAmount <= 0 {
		refundAmount = total // full refund
	}
	if refundAmount <= 0 || refundAmount > total+0.001 {
		return 0, ErrInvalidRefundAmount
	}
	isFull := refundAmount >= total-0.001

	var platformWalletID int64
	if err := tx.Get(&platformWalletID, `
		SELECT wallet_id FROM wallets
		WHERE user_id = (SELECT (value)::bigint FROM tconfig WHERE key = 'platform_user_id')
	`); err != nil {
		return 0, err
	}

	if isFull {
		// factory: release held receivable (SC/PT)
		if _, err := tx.Exec(`
			UPDATE wallets w SET pending_fund = GREATEST(w.pending_fund - t.amount, 0)
			FROM transactions t
			WHERE t.order_id = $1 AND t.type = 'SC' AND t.status = 'PT' AND w.wallet_id = t.wallet_id
		`, orderID); err != nil {
			return 0, err
		}
		// platform: release escrow-in hold (EI/PT)
		if _, err := tx.Exec(`
			UPDATE wallets w SET pending_fund = GREATEST(w.pending_fund - t.amount, 0)
			FROM transactions t
			WHERE t.order_id = $1 AND t.type = 'EI' AND t.status = 'PT' AND w.wallet_id = t.wallet_id
		`, orderID); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(`
			UPDATE transactions SET status = 'RF', updated_at = NOW()
			WHERE order_id = $1 AND type IN ('SC','EI','CM','BU') AND status = 'PT'
		`, orderID); err != nil {
			return 0, err
		}
	} else {
		// partial: draw the refund from the platform's escrow-in hold only
		if _, err := tx.Exec(`
			UPDATE wallets SET pending_fund = GREATEST(pending_fund - $2, 0) WHERE wallet_id = $1
		`, platformWalletID, refundAmount); err != nil {
			return 0, err
		}
	}

	// record the outgoing refund payout (money leaves Tryly's bank → customer)
	if _, err := tx.Exec(`
		INSERT INTO transactions (wallet_id, order_id, type, status, amount, slip_url)
		VALUES ($1, $2, 'RF', 'ST', $3, $4)
	`, platformWalletID, orderID, refundAmount, slipURL); err != nil {
		return 0, err
	}

	if _, err := tx.Exec(`
		UPDATE disputes
		SET status = 'RF', resolution = NULLIF($2,''), refund_amount = $3, refund_slip_url = $4,
		    resolved_by = $5, resolved_at = NOW(), updated_at = NOW()
		WHERE dispute_id = $1
	`, disputeID, resolution, refundAmount, slipURL, resolvedBy); err != nil {
		return 0, err
	}
	// full refund → terminal 'RF' (คืนเงินแล้ว) — distinct from a plain cancellation
	// so the FE can label it. partial refund restores the pre-dispute status
	// (order keeps going).
	nextOrderStatus := "PD"
	if priorStatus != nil && *priorStatus != "" {
		nextOrderStatus = *priorStatus
	}
	if isFull {
		nextOrderStatus = "RF"
	}
	if _, err := tx.Exec(`
		UPDATE orders SET status = $2, updated_at = NOW() WHERE order_id = $1
	`, orderID, nextOrderStatus); err != nil {
		return 0, err
	}
	return refundAmount, tx.Commit()
}
