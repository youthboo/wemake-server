package domain

import (
	"encoding/json"
	"time"
)

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type AdminFactoryFilter struct {
	ApprovalStatus string
	Search         string
	Page           int
	PageSize       int
	IsVerified     *bool
}

type AdminFactoryListItem struct {
	FactoryID       int64      `db:"factory_id" json:"factory_id"`
	FactoryName     string     `db:"factory_name" json:"factory_name"`
	Email           string     `db:"email" json:"email"`
	Phone           *string    `db:"phone" json:"phone,omitempty"`
	TaxID           *string    `db:"tax_id" json:"tax_id,omitempty"`
	FactoryTypeName *string    `db:"factory_type_name" json:"factory_type_name,omitempty"`
	ProvinceName    *string    `db:"province_name" json:"province_name,omitempty"`
	ApprovalStatus  string     `db:"approval_status" json:"approval_status"`
	IsVerified      bool       `db:"is_verified" json:"is_verified"`
	SubmittedAt     *time.Time `db:"submitted_at" json:"submitted_at,omitempty"`
	VerifiedAt      *time.Time `db:"verified_at" json:"verified_at,omitempty"`
	VerifiedBy      *int64     `db:"verified_by" json:"verified_by,omitempty"`
	RejectionReason *string    `db:"rejection_reason" json:"rejection_reason,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

type AdminFactoryStats struct {
	TotalOrders     int64 `db:"total_orders" json:"total_orders"`
	TotalQuotations int64 `db:"total_quotations" json:"total_quotations"`
	TotalShowcases  int64 `db:"total_showcases" json:"total_showcases"`
}

type AdminFactoryDetail struct {
	AdminFactoryListItem
	FactoryTypeID       int64                       `db:"factory_type_id" json:"factory_type_id"`
	Specialization      *string                     `db:"specialization" json:"specialization,omitempty"`
	ProvinceID          *int64                      `db:"province_id" json:"province_id,omitempty"`
	ImageURL            *string                     `db:"image_url" json:"image_url,omitempty"`
	Description         *string                     `db:"description" json:"description,omitempty"`
	Categories          []FactoryProfileCategory    `json:"categories"`
	SubCategories       []FactoryProfileSubCategory `json:"sub_categories"`
	Certificates        []FactoryProfileCertificate `json:"certificates"`
	Stats               AdminFactoryStats           `json:"stats"`
	CommissionOverride  *CommissionRule             `json:"commission_override,omitempty"`
	IsCommissionExempt  bool                        `json:"is_commission_exempt"`
}

type AdminDashboardSummary struct {
	Period      string                  `json:"period"`
	DateFrom    string                  `json:"date_from"`
	DateTo      string                  `json:"date_to"`
	Revenue     AdminDashboardRevenue   `json:"revenue"`
	Orders      AdminDashboardOrders    `json:"orders"`
	RFQs        AdminDashboardRFQs      `json:"rfqs"`
	Factories   AdminDashboardFactories `json:"factories"`
	Customers   AdminDashboardCustomers `json:"customers"`
	Settlements AdminDashboardMoney     `json:"settlements"`
	Withdrawals AdminDashboardPending   `json:"withdrawals"`
}

type AdminDashboardRevenue struct {
	GrossOrderValue    float64 `db:"gross_order_value" json:"gross_order_value"`
	TotalVATCollected  float64 `db:"total_vat_collected" json:"total_vat_collected"`
	PlatformCommission float64 `db:"platform_commission" json:"platform_commission"`
	FactoryNetPayable  float64 `db:"factory_net_payable" json:"factory_net_payable"`
}

type AdminDashboardOrders struct {
	Total     int64 `db:"total" json:"total"`
	Completed int64 `db:"completed" json:"completed"`
	Active    int64 `db:"active" json:"active"`
	Cancelled int64 `db:"cancelled" json:"cancelled"`
	Disputed  int64 `db:"disputed" json:"disputed"`
}

type AdminDashboardRFQs struct {
	Total  int64 `db:"total" json:"total"`
	Open   int64 `db:"open" json:"open"`
	Closed int64 `db:"closed" json:"closed"`
}

type AdminDashboardFactories struct {
	TotalRegistered int64 `db:"total_registered" json:"total_registered"`
	PendingApproval int64 `db:"pending_approval" json:"pending_approval"`
	Approved        int64 `db:"approved" json:"approved"`
	Rejected        int64 `db:"rejected" json:"rejected"`
	Suspended       int64 `db:"suspended" json:"suspended"`
}

type AdminDashboardCustomers struct {
	Total int64 `db:"total" json:"total"`
}

type AdminDashboardMoney struct {
	PendingAmount   float64 `db:"pending_amount" json:"pending_amount,omitempty"`
	CompletedAmount float64 `db:"completed_amount" json:"completed_amount,omitempty"`
}

type AdminDashboardPending struct {
	PendingCount  int64   `db:"pending_count" json:"pending_count"`
	PendingAmount float64 `db:"pending_amount" json:"pending_amount"`
}

type RevenueChartPoint struct {
	Date               string  `db:"bucket" json:"date"`
	GrossOrderValue    float64 `db:"gross_order_value" json:"gross_order_value"`
	PlatformCommission float64 `db:"platform_commission" json:"platform_commission"`
	VATCollected       float64 `db:"vat_collected" json:"vat_collected"`
	OrderCount         int64   `db:"order_count" json:"order_count"`
}

type TopFactoryRow struct {
	FactoryID          int64    `db:"factory_id" json:"factory_id"`
	FactoryName        string   `db:"factory_name" json:"factory_name"`
	TotalOrders        int64    `db:"total_orders" json:"total_orders"`
	CompletedOrders    int64    `db:"completed_orders" json:"completed_orders"`
	GrossRevenue       float64  `db:"gross_revenue" json:"gross_revenue"`
	PlatformCommission float64  `db:"platform_commission" json:"platform_commission"`
	AvgRating          *float64 `db:"avg_rating" json:"avg_rating,omitempty"`
}

type AdminRFQFilter struct {
	Status     string
	UserID     *int64
	CategoryID *int64
	DateFrom   *time.Time
	DateTo     *time.Time
	Search     string
	Page       int
	PageSize   int
}

type AdminRFQListItem struct {
	RFQID            int64      `db:"rfq_id" json:"rfq_id"`
	Title            string     `db:"title" json:"title"`
	UserID           int64      `db:"user_id" json:"user_id"`
	CustomerName     string     `db:"customer_name" json:"customer_name"`
	CustomerEmail    string     `db:"customer_email" json:"customer_email"`
	CategoryName     string     `db:"category_name" json:"category_name"`
	SubCategoryName  *string    `db:"sub_category_name" json:"sub_category_name,omitempty"`
	Quantity         int64      `db:"quantity" json:"quantity"`
	Status           string     `db:"status" json:"status"`
	QuotationCount   int64      `db:"quotation_count" json:"quotation_count"`
	TargetUnitPrice  *float64   `db:"target_unit_price" json:"target_unit_price,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
}

type AdminRFQDetail struct {
	RFQ             *RFQ                    `json:"rfq"`
	CustomerName    string                  `json:"customer_name"`
	CustomerEmail   string                  `json:"customer_email"`
	CustomerPhone   *string                 `json:"customer_phone,omitempty"`
	QuotationCount  int64                   `json:"quotation_count"`
}

type AdminOrderFilter struct {
	Status    string
	FactoryID *int64
	UserID    *int64
	DateFrom  *time.Time
	DateTo    *time.Time
	Search    string
	Page      int
	PageSize  int
}

type AdminOrderListItem struct {
	OrderID                  int64      `db:"order_id" json:"order_id"`
	QuoteID                  int64      `db:"quote_id" json:"quote_id"`
	RFQID                    int64      `db:"rfq_id" json:"rfq_id"`
	RFQTitle                 string     `db:"rfq_title" json:"rfq_title"`
	FactoryID                int64      `db:"factory_id" json:"factory_id"`
	FactoryName              string     `db:"factory_name" json:"factory_name"`
	UserID                   int64      `db:"user_id" json:"user_id"`
	CustomerName             string     `db:"customer_name" json:"customer_name"`
	Status                   string     `db:"status" json:"status"`
	TotalAmount              float64    `db:"total_amount" json:"total_amount"`
	PlatformCommissionAmount float64    `db:"platform_commission_amount" json:"platform_commission_amount"`
	VATAmount                float64    `db:"vat_amount" json:"vat_amount"`
	FactoryNetReceivable     float64    `db:"factory_net_receivable" json:"factory_net_receivable"`
	PaymentType              *string    `db:"payment_type" json:"payment_type,omitempty"`
	EstimatedDelivery        *time.Time `db:"estimated_delivery" json:"estimated_delivery,omitempty"`
	CreatedAt                time.Time  `db:"created_at" json:"created_at"`
}

type AdminOrderFinance struct {
	PlatformCommissionRate   float64 `db:"platform_commission_rate" json:"platform_commission_rate"`
	PlatformCommissionAmount float64 `db:"platform_commission_amount" json:"platform_commission_amount"`
	VATRate                  float64 `db:"vat_rate" json:"vat_rate"`
	VATAmount                float64 `db:"vat_amount" json:"vat_amount"`
	FactoryNetReceivable     float64 `db:"factory_net_receivable" json:"factory_net_receivable"`
	GrandTotal               float64 `db:"grand_total" json:"grand_total"`
}

type AdminOrderDetailResponse struct {
	*OrderDetailResponse
	AdminFinance AdminOrderFinance `json:"admin_finance"`
}

type AdminWithdrawalListItem struct {
	RequestID      int64      `db:"request_id" json:"request_id"`
	FactoryID      int64      `db:"factory_id" json:"factory_id"`
	FactoryName    string     `db:"factory_name" json:"factory_name"`
	Amount         float64    `db:"amount" json:"amount"`
	BankName       string     `db:"bank_name" json:"bank_name"`
	BankAccountNo  string     `db:"bank_account_no" json:"bank_account_no"`
	AccountName    string     `db:"account_name" json:"account_name"`
	Status         string     `db:"status" json:"status"`
	ProcessedAt    *time.Time `db:"processed_at" json:"processed_at,omitempty"`
	Note           *string    `db:"note" json:"note,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

type AdminDisputeListItem struct {
	DisputeID     int64      `db:"dispute_id" json:"dispute_id"`
	OrderID       int64      `db:"order_id" json:"order_id"`
	RFQTitle      string     `db:"rfq_title" json:"rfq_title"`
	FactoryName   string     `db:"factory_name" json:"factory_name"`
	CustomerName  string     `db:"customer_name" json:"customer_name"`
	OpenedBy      int64      `db:"opened_by" json:"opened_by"`
	Reason        string     `db:"reason" json:"reason"`
	Status        string     `db:"status" json:"status"`
	Resolution    *string    `db:"resolution" json:"resolution,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	ResolvedAt    *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
}

type CommissionRule struct {
	RuleID        int64      `db:"rule_id" json:"rule_id"`
	FactoryID     *int64     `db:"factory_id" json:"factory_id,omitempty"`
	FactoryName   *string    `db:"factory_name" json:"factory_name,omitempty"`
	RatePercent   float64    `db:"rate_percent" json:"rate_percent"`
	EffectiveFrom time.Time  `db:"effective_from" json:"effective_from"`
	EffectiveTo   *time.Time `db:"effective_to" json:"effective_to,omitempty"`
	Note          *string    `db:"note" json:"note,omitempty"`
	CreatedBy     int64      `db:"created_by" json:"created_by"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

type CommissionExemption struct {
	ExemptionID int64      `db:"exemption_id" json:"exemption_id"`
	FactoryID   int64      `db:"factory_id" json:"factory_id"`
	FactoryName *string    `db:"factory_name" json:"factory_name,omitempty"`
	Reason      string     `db:"reason" json:"reason"`
	ExpiresAt   *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	CreatedBy   int64      `db:"created_by" json:"created_by"`
	RevokedBy   *int64     `db:"revoked_by" json:"revoked_by,omitempty"`
	RevokedAt   *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	IsActive    bool       `db:"is_active" json:"is_active"`
}

type AdminAuditLog struct {
	LogID       int64           `db:"log_id" json:"log_id"`
	ActorID     int64           `db:"actor_id" json:"actor_id"`
	ActorEmail  *string         `db:"actor_email" json:"actor_email,omitempty"`
	Action      string          `db:"action" json:"action"`
	TargetType  string          `db:"target_type" json:"target_type"`
	TargetID    string          `db:"target_id" json:"target_id"`
	Payload     json.RawMessage `db:"payload" json:"payload,omitempty"`
	IPAddress   *string         `db:"ip_address" json:"ip_address,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}

type AdminAuditFilter struct {
	ActorID    *int64
	Action     string
	TargetType string
	DateFrom   *time.Time
	DateTo     *time.Time
	Page       int
	PageSize   int
}

type AdminUserListItem struct {
	UserID      int64      `db:"user_id" json:"user_id"`
	Email       string     `db:"email" json:"email"`
	Role        string     `db:"role" json:"role"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	DisplayName *string    `db:"display_name" json:"display_name,omitempty"`
	Department  *string    `db:"department" json:"department,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

type AdminProfile struct {
	UserID      int64      `db:"user_id" json:"user_id"`
	DisplayName string     `db:"display_name" json:"display_name"`
	Department  *string    `db:"department" json:"department,omitempty"`
	CreatedBy   *int64     `db:"created_by" json:"created_by,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

// ─── Customer admin models ────────────────────────────────────────────────────

type AdminCustomerListItem struct {
	UserID        int64   `db:"user_id"        json:"user_id"`
	Email         string  `db:"email"          json:"email"`
	FirstName     string  `db:"first_name"     json:"first_name"`
	LastName      string  `db:"last_name"      json:"last_name"`
	Phone         string  `db:"phone"          json:"phone,omitempty"`
	IsActive      bool    `db:"is_active"      json:"is_active"`
	TotalOrders   int     `db:"total_orders"   json:"total_orders"`
	TotalSpend    float64 `db:"total_spend"    json:"total_spend"`
	WalletBalance float64 `db:"wallet_balance" json:"wallet_balance"`
	CreatedAt     string  `db:"created_at"     json:"created_at"`
}

type AdminCustomerDetail struct {
	UserID      int64   `db:"user_id"      json:"user_id"`
	Email       string  `db:"email"        json:"email"`
	FirstName   string  `db:"first_name"   json:"first_name"`
	LastName    string  `db:"last_name"    json:"last_name"`
	Phone       string  `db:"phone"        json:"phone,omitempty"`
	Address     string  `db:"address"      json:"address,omitempty"`
	IsActive    bool    `db:"is_active"    json:"is_active"`
	TotalOrders int     `db:"total_orders" json:"total_orders"`
	TotalSpend  float64 `db:"total_spend"  json:"total_spend"`
	CreatedAt   string  `db:"created_at"   json:"created_at"`
	WalletID    *int64  `db:"wallet_id"    json:"wallet_id,omitempty"`
	GoodFund    float64 `db:"good_fund"    json:"good_fund"`
	PendingFund float64 `db:"pending_fund" json:"pending_fund"`
}

type AdminWalletTxItem struct {
	TxID      string   `db:"tx_id"     json:"tx_id"`
	WalletID  int64    `db:"wallet_id" json:"wallet_id"`
	OrderID   *int64   `db:"order_id"  json:"order_id,omitempty"`
	Type      string   `db:"type"      json:"type"`
	Amount    float64  `db:"amount"    json:"amount"`
	Status    string   `db:"status"    json:"status"`
	CreatedAt string   `db:"created_at" json:"created_at"`
}

type AdminCustomerWallet struct {
	WalletID     *int64              `json:"wallet_id,omitempty"`
	UserID       int64               `json:"user_id"`
	GoodFund     float64             `json:"good_fund"`
	PendingFund  float64             `json:"pending_fund"`
	Total        float64             `json:"total"`
	Transactions []AdminWalletTxItem `json:"transactions"`
}

type AdminTopCustomer struct {
	UserID      int64   `db:"user_id"      json:"user_id"`
	FirstName   string  `db:"first_name"   json:"first_name"`
	LastName    string  `db:"last_name"    json:"last_name"`
	Email       string  `db:"email"        json:"email"`
	TotalOrders int     `db:"total_orders" json:"total_orders"`
	TotalSpend  float64 `db:"total_spend"  json:"total_spend"`
}

// ─── Settlement admin models ──────────────────────────────────────────────────

type AdminSettlementListItem struct {
	SettlementID int64   `db:"settlement_id" json:"settlement_id"`
	FactoryID    int64   `db:"factory_id"    json:"factory_id"`
	OrderID      int64   `db:"order_id"      json:"order_id"`
	Amount       float64 `db:"amount"        json:"amount"`
	Status       string  `db:"status"        json:"status"`
	CreatedAt    string  `db:"created_at"    json:"created_at"`
	UpdatedAt    string  `db:"updated_at"    json:"updated_at,omitempty"`
}

// AdminCustomerOrderItem is a lightweight order row for customer detail page
type AdminCustomerOrderItem struct {
	OrderID     int64   `db:"order_id"     json:"order_id"`
	RFQID       int64   `db:"rfq_id"       json:"rfq_id"`
	FactoryID   int64   `db:"factory_id"   json:"factory_id"`
	FactoryName string  `db:"factory_name" json:"factory_name"`
	GrandTotal  float64 `db:"grand_total"  json:"grand_total"`
	Status      string  `db:"status"       json:"status"`
	CreatedAt   string  `db:"created_at"   json:"created_at"`
}
