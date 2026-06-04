package factory

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// BankAccount maps to factory_bank_accounts table.
type BankAccount struct {
	AccountID     int64  `db:"account_id"     json:"account_id"`
	FactoryID     int64  `db:"factory_id"     json:"factory_id"`
	BankName      string `db:"bank_name"      json:"bank_name"`
	AccountNumber string `db:"account_number" json:"account_number"`
	AccountName   string `db:"account_name"   json:"account_name"`
	IsDefault     bool   `db:"is_default"     json:"is_default"`
	CreatedAt     string `db:"created_at"     json:"created_at"`
	UpdatedAt     string `db:"updated_at"     json:"updated_at"`
}

type BankAccountRepository struct {
	db *sqlx.DB
}

func NewBankAccountRepository(db *sqlx.DB) *BankAccountRepository {
	return &BankAccountRepository{db: db}
}

// ListByFactory returns all bank accounts for a factory, ordered by is_default DESC.
func (r *BankAccountRepository) ListByFactory(factoryID int64) ([]BankAccount, error) {
	var items []BankAccount
	err := r.db.Select(&items, `
		SELECT account_id, factory_id, bank_name, account_number, account_name,
		       is_default, created_at::text AS created_at, updated_at::text AS updated_at
		FROM factory_bank_accounts
		WHERE factory_id = $1
		ORDER BY is_default DESC, account_id ASC
	`, factoryID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []BankAccount{}
	}
	return items, nil
}

// GetDefault returns the default bank account for a factory.
func (r *BankAccountRepository) GetDefault(factoryID int64) (*BankAccount, error) {
	var item BankAccount
	err := r.db.Get(&item, `
		SELECT account_id, factory_id, bank_name, account_number, account_name,
		       is_default, created_at::text AS created_at, updated_at::text AS updated_at
		FROM factory_bank_accounts
		WHERE factory_id = $1 AND is_default = true
		LIMIT 1
	`, factoryID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByID returns a bank account by its ID.
func (r *BankAccountRepository) GetByID(accountID int64) (*BankAccount, error) {
	var item BankAccount
	err := r.db.Get(&item, `
		SELECT account_id, factory_id, bank_name, account_number, account_name,
		       is_default, created_at::text AS created_at, updated_at::text AS updated_at
		FROM factory_bank_accounts
		WHERE account_id = $1
	`, accountID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CountByFactory returns the number of bank accounts for a factory.
func (r *BankAccountRepository) CountByFactory(factoryID int64) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM factory_bank_accounts WHERE factory_id = $1`, factoryID)
	return count, err
}

// Create inserts a new bank account. If is_default=true, unsets other defaults first.
func (r *BankAccountRepository) Create(factoryID int64, bankName, accountNumber, accountName string, isDefault bool) (*BankAccount, error) {
	if isDefault {
		r.db.Exec(`UPDATE factory_bank_accounts SET is_default = false, updated_at = NOW() WHERE factory_id = $1 AND is_default = true`, factoryID)
	}

	var item BankAccount
	err := r.db.Get(&item, `
		INSERT INTO factory_bank_accounts (factory_id, bank_name, account_number, account_name, is_default)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING account_id, factory_id, bank_name, account_number, account_name, is_default,
		          created_at::text AS created_at, updated_at::text AS updated_at
	`, factoryID, bankName, accountNumber, accountName, isDefault)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Update patches a bank account. If is_default=true, unsets other defaults.
func (r *BankAccountRepository) Update(accountID int64, factoryID int64, bankName, accountNumber, accountName *string, isDefault *bool) (*BankAccount, error) {
	if isDefault != nil && *isDefault {
		r.db.Exec(`UPDATE factory_bank_accounts SET is_default = false, updated_at = NOW() WHERE factory_id = $1 AND is_default = true AND account_id != $2`, factoryID, accountID)
	}

	var item BankAccount
	err := r.db.Get(&item, `
		UPDATE factory_bank_accounts
		SET bank_name       = COALESCE($2, bank_name),
		    account_number  = COALESCE($3, account_number),
		    account_name    = COALESCE($4, account_name),
		    is_default      = COALESCE($5, is_default),
		    updated_at      = NOW()
		WHERE account_id = $1
		RETURNING account_id, factory_id, bank_name, account_number, account_name, is_default,
		          created_at::text AS created_at, updated_at::text AS updated_at
	`, accountID, bankName, accountNumber, accountName, isDefault)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Delete removes a bank account.
func (r *BankAccountRepository) Delete(accountID int64) error {
	res, err := r.db.Exec(`DELETE FROM factory_bank_accounts WHERE account_id = $1`, accountID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
