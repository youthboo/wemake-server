package auth

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
)

type AuthRepository struct {
	db *sqlx.DB
}

func NewAuthRepository(db *sqlx.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) GetUserByID(userID int64) (*domain.User, error) {
	var user domain.User
	query := `
		SELECT user_id, role, email, phone, NULL::text AS avatar_url, NULL::text AS bio, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE user_id = $1
	`
	if err := r.db.Get(&user, query, userID); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	query := `
		SELECT user_id, role, email, phone, NULL::text AS avatar_url, NULL::text AS bio, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	if err := r.db.Get(&user, query, email); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) CreateCustomerUser(user *domain.User, customer *domain.CustomerProfile) error {
	return helper.WithTx(nil, r.db, func(tx *sqlx.Tx) error {
		const userInsert = `
			INSERT INTO users (role, email, phone, password_hash, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING user_id
		`
		if err := tx.QueryRow(
			userInsert,
			user.Role,
			user.Email,
			user.Phone,
			user.PasswordHash,
			user.IsActive,
			user.CreatedAt,
			user.UpdatedAt,
		).Scan(&user.UserID); err != nil {
			return err
		}

		const customerInsert = `
			INSERT INTO customers (user_id, first_name, last_name)
			VALUES ($1, $2, $3)
		`
		if _, err := tx.Exec(customerInsert, user.UserID, customer.FirstName, customer.LastName); err != nil {
			return err
		}

		// Create wallet for new customer (balance starts at 0)
		if _, err := tx.Exec(`
			INSERT INTO wallets (user_id, good_fund, pending_fund)
			VALUES ($1, 0, 0)
		`, user.UserID); err != nil {
			return err
		}
		return nil
	})
}

func (r *AuthRepository) CreateFactoryUser(user *domain.User, factory *domain.FactoryProfile, categoryIDs []int64, subCategoryIDs []int64, certID int64, documentURL string, certNumber string, certExpireDate string, addr *domain.Address) error {
	return helper.WithTx(nil, r.db, func(tx *sqlx.Tx) error {
		const userInsert = `
			INSERT INTO users (role, email, phone, password_hash, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING user_id
		`
		if err := tx.QueryRow(
			userInsert,
			user.Role, user.Email, user.Phone, user.PasswordHash, user.IsActive, user.CreatedAt, user.UpdatedAt,
		).Scan(&user.UserID); err != nil {
			return err
		}

		var provinceID sql.NullInt64
		if factory.ProvinceID != nil && *factory.ProvinceID > 0 {
			provinceID = sql.NullInt64{Int64: *factory.ProvinceID, Valid: true}
		}
		var factoryID int64
		if err := tx.QueryRow(`
			INSERT INTO factory_profiles
				(user_id, factory_name, factory_type_id, tax_id, province_id, review_count, completed_orders, approval_status, submitted_at)
			VALUES ($1, $2, $3, $4, $5, 0, 0, 'PE', NOW())
			RETURNING user_id
		`, user.UserID, factory.FactoryName, factory.FactoryTypeID, factory.TaxID, provinceID).Scan(&factoryID); err != nil {
			return err
		}

		for _, catID := range categoryIDs {
			if _, err := tx.Exec(
				`INSERT INTO map_factory_categories (factory_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				factoryID, catID,
			); err != nil {
				return err
			}
		}
		for _, subID := range subCategoryIDs {
			if _, err := tx.Exec(
				`INSERT INTO map_factory_sub_categories (factory_id, sub_category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				factoryID, subID,
			); err != nil {
				return err
			}
		}

		// Create wallet for new factory (balance starts at 0)
		if _, err := tx.Exec(`
			INSERT INTO wallets (user_id, good_fund, pending_fund)
			VALUES ($1, 0, 0)
		`, user.UserID); err != nil {
			return err
		}

		// Insert certificate if provided
		if certID > 0 && documentURL != "" {
			var expireDateVal sql.NullTime
			if certExpireDate != "" {
				t, err := time.Parse("2006-01-02", certExpireDate)
				if err == nil {
					expireDateVal = sql.NullTime{Time: t, Valid: true}
				}
			}
			certNumberVal := sql.NullString{}
			if certNumber != "" {
				certNumberVal = sql.NullString{String: certNumber, Valid: true}
			}
			if _, err := tx.Exec(`
				INSERT INTO map_factory_certificates
					(factory_id, cert_id, document_url, expire_date, cert_number, verify_status)
				VALUES ($1, $2, $3, $4, $5, 'PE')
			`, factoryID, certID, documentURL, expireDateVal, certNumberVal); err != nil {
				return err
			}
		}

		// Insert default address if provided
		if addr != nil && addr.ProvinceID > 0 {
			if _, err := tx.Exec(`
				INSERT INTO addresses (user_id, address_type, address_detail, sub_district_id, district_id, province_id, zip_code, is_default)
				VALUES ($1, 'B', $2, $3, $4, $5, $6, true)
			`, user.UserID, addr.AddressDetail, addr.SubDistrictID, addr.DistrictID, addr.ProvinceID, addr.ZipCode); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *AuthRepository) CreateAdminUser(user *domain.User, profile *domain.AdminProfile) error {
	return helper.WithTx(nil, r.db, func(tx *sqlx.Tx) error {
		if err := tx.QueryRow(`
			INSERT INTO users (role, email, phone, password_hash, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING user_id
		`, user.Role, user.Email, user.Phone, user.PasswordHash, user.IsActive, user.CreatedAt, user.UpdatedAt).Scan(&user.UserID); err != nil {
			return err
		}

		if profile != nil {
			profile.UserID = user.UserID
			if err := tx.QueryRow(`
				INSERT INTO admin_profiles (user_id, display_name, department, created_by)
				VALUES ($1, $2, $3, $4)
				RETURNING created_at
			`, profile.UserID, profile.DisplayName, domainutil.Nullable(profile.Department), domainutil.Nullable(profile.CreatedBy)).Scan(&profile.CreatedAt); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *AuthRepository) ListAdminUsers() ([]domain.AdminUserListItem, error) {
	var items []domain.AdminUserListItem
	err := r.db.Select(&items, `
		SELECT
			u.user_id,
			u.email,
			u.role,
			u.is_active,
			ap.display_name,
			ap.department,
			u.created_at
		FROM users u
		LEFT JOIN admin_profiles ap ON ap.user_id = u.user_id
		WHERE u.role IN ('AM', 'AD', 'SA')
		ORDER BY u.created_at DESC, u.user_id DESC
	`)
	return items, err
}

func (r *AuthRepository) UpgradeToCustomer(userID int64, firstName, lastName string) error {
	return helper.WithTx(nil, r.db, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(
			`UPDATE users SET role = 'CT', updated_at = NOW() WHERE user_id = $1`,
			userID,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO customers (user_id, first_name, last_name) VALUES ($1, $2, $3)`,
			userID, firstName, lastName,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO wallets (user_id, good_fund, pending_fund) VALUES ($1, 0, 0) ON CONFLICT DO NOTHING`,
			userID,
		); err != nil {
			return err
		}
		return nil
	})
}

// UpgradeToFactory upgrades an existing CT user to FT within one transaction.
// The caller must confirm the user exists and is CT before calling this.
func (r *AuthRepository) UpgradeToFactory(userID int64, factory *domain.FactoryProfile, categoryIDs []int64, subCategoryIDs []int64, certID int64, documentURL string, certNumber string, certExpireDate string, addr *domain.Address) error {
	return helper.WithTx(nil, r.db, func(tx *sqlx.Tx) error {
		// 1. Update user role to FT
		if _, err := tx.Exec(
			`UPDATE users SET role = 'FT', updated_at = NOW() WHERE user_id = $1`,
			userID,
		); err != nil {
			return err
		}

		// 2. Create factory profile (factory_profiles.user_id = PK)
		var provinceID sql.NullInt64
		if factory.ProvinceID != nil && *factory.ProvinceID > 0 {
			provinceID = sql.NullInt64{Int64: *factory.ProvinceID, Valid: true}
		}
		var factoryID int64
		if err := tx.QueryRow(`
			INSERT INTO factory_profiles
				(user_id, factory_name, factory_type_id, tax_id, province_id, review_count, completed_orders, approval_status, submitted_at)
			VALUES ($1, $2, $3, $4, $5, 0, 0, 'PE', NOW())
			RETURNING user_id
		`, userID, factory.FactoryName, factory.FactoryTypeID, factory.TaxID, provinceID).Scan(&factoryID); err != nil {
			return err
		}

		// 3. Categories
		for _, catID := range categoryIDs {
			if _, err := tx.Exec(
				`INSERT INTO map_factory_categories (factory_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				factoryID, catID,
			); err != nil {
				return err
			}
		}
		for _, subID := range subCategoryIDs {
			if _, err := tx.Exec(
				`INSERT INTO map_factory_sub_categories (factory_id, sub_category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				factoryID, subID,
			); err != nil {
				return err
			}
		}

		// 4. Certificate
		if certID > 0 && documentURL != "" {
			var expireDateVal sql.NullTime
			if certExpireDate != "" {
				t, err := time.Parse("2006-01-02", certExpireDate)
				if err == nil {
					expireDateVal = sql.NullTime{Time: t, Valid: true}
				}
			}
			certNumberVal := sql.NullString{}
			if certNumber != "" {
				certNumberVal = sql.NullString{String: certNumber, Valid: true}
			}
			if _, err := tx.Exec(`
				INSERT INTO map_factory_certificates
					(factory_id, cert_id, document_url, expire_date, cert_number, verify_status)
				VALUES ($1, $2, $3, $4, $5, 'PE')
			`, factoryID, certID, documentURL, expireDateVal, certNumberVal); err != nil {
				return err
			}
		}

		// 5. CT already has a wallet — skip wallet creation

		// 6. Insert default address if provided
		if addr != nil && addr.ProvinceID > 0 {
			if _, err := tx.Exec(`
				INSERT INTO addresses (user_id, address_type, address_detail, sub_district_id, district_id, province_id, zip_code, is_default)
				VALUES ($1, 'B', $2, $3, $4, $5, $6, true)
				ON CONFLICT DO NOTHING
			`, userID, addr.AddressDetail, addr.SubDistrictID, addr.DistrictID, addr.ProvinceID, addr.ZipCode); err != nil {
				return err
			}
		}

		return nil
	})
}

// HasProfile checks whether the user has a customer or factory profile row.
func (r *AuthRepository) HasProfile(userID int64, role string) (bool, error) {
	var query string
	switch role {
	case "CT":
		query = `SELECT EXISTS(SELECT 1 FROM customers WHERE user_id = $1)`
	case "FT":
		query = `SELECT EXISTS(SELECT 1 FROM factory_profiles WHERE user_id = $1)`
	default:
		return false, nil
	}
	var exists bool
	if err := r.db.Get(&exists, query, userID); err != nil {
		return false, err
	}
	return exists, nil
}

// UpdateRole changes the active role on the users table.
func (r *AuthRepository) UpdateRole(userID int64, role string) error {
	_, err := r.db.Exec(`UPDATE users SET role = $1, updated_at = NOW() WHERE user_id = $2`, role, userID)
	return err
}

func (r *AuthRepository) UpdateLoginTimestamp(userID int64, loginAt time.Time) error {
	query := "UPDATE users SET updated_at = $1 WHERE user_id = $2"
	_, err := r.db.Exec(query, loginAt, userID)
	return err
}

func (r *AuthRepository) CreatePasswordResetToken(token *domain.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	return r.db.QueryRow(query, token.UserID, token.Token, token.ExpiresAt, token.CreatedAt).Scan(&token.ID)
}

func (r *AuthRepository) GetValidPasswordResetToken(token string) (*domain.PasswordResetToken, error) {
	var resetToken domain.PasswordResetToken
	query := `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token = $1 AND used_at IS NULL AND expires_at > NOW()
	`
	if err := r.db.Get(&resetToken, query, token); err != nil {
		return nil, err
	}
	return &resetToken, nil
}

func (r *AuthRepository) ResetPassword(userID int64, tokenID int64, passwordHash string, now time.Time) error {
	return helper.WithTx(nil, r.db, func(tx *sqlx.Tx) error {
		const updateUser = "UPDATE users SET password_hash = $1, updated_at = $2 WHERE user_id = $3"
		if _, err := tx.Exec(updateUser, passwordHash, now, userID); err != nil {
			return err
		}

		const markToken = "UPDATE password_reset_tokens SET used_at = $1 WHERE id = $2"
		if _, err := tx.Exec(markToken, now, tokenID); err != nil {
			return err
		}
		return nil
	})
}

func (r *AuthRepository) UpdatePassword(userID int64, passwordHash string, now time.Time) error {
	_, err := r.db.Exec(`UPDATE users SET password_hash = $1, updated_at = $2 WHERE user_id = $3`, passwordHash, now, userID)
	return err
}
