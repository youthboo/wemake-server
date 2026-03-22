package domain

import "time"

type Category struct {
	CategoryID int64  `db:"category_id" json:"category_id"`
	Name       string `db:"name" json:"name"`
}

type Unit struct {
	UnitID int64  `db:"unit_id" json:"unit_id"`
	Name   string `db:"name" json:"name"`
}

type Address struct {
	AddressID     int64  `db:"address_id" json:"address_id"`
	UserID        int64  `db:"user_id" json:"user_id"`
	AddressType   string `db:"address_type" json:"address_type"`
	AddressDetail string `db:"address_detail" json:"address_detail"`
	SubDistrictID int64  `db:"sub_district_id" json:"sub_district_id"`
	DistrictID    int64  `db:"district_id" json:"district_id"`
	ProvinceID    int64  `db:"province_id" json:"province_id"`
	ZipCode       string `db:"zip_code" json:"zip_code"`
	IsDefault     bool   `db:"is_default" json:"is_default"`
}

type Wallet struct {
	WalletID    int64   `db:"wallet_id" json:"wallet_id"`
	UserID      int64   `db:"user_id" json:"user_id"`
	GoodFund    float64 `db:"good_fund" json:"good_fund"`
	PendingFund float64 `db:"pending_fund" json:"pending_fund"`
}

type RFQ struct {
	RFQID          int64     `db:"rfq_id" json:"rfq_id"`
	UserID         int64     `db:"user_id" json:"user_id"`
	CategoryID     int64     `db:"category_id" json:"category_id"`
	Title          string    `db:"title" json:"title"`
	Quantity       int64     `db:"quantity" json:"quantity"`
	UnitID         int64     `db:"unit_id" json:"unit_id"`
	BudgetPerPiece float64   `db:"budget_per_piece" json:"budget_per_piece"`
	Details        string    `db:"details" json:"details"`
	AddressID      int64     `db:"address_id" json:"address_id"`
	Status         string    `db:"status" json:"status"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

type RFQImage struct {
	ImageID  string `db:"image_id" json:"image_id"`
	RFQID    int64  `db:"rfq_id" json:"rfq_id"`
	ImageURL string `db:"image_url" json:"image_url"`
}
