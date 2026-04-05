package repository

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type AuthRepository struct {
	db *sqlx.DB
}

func NewAuthRepository(db *sqlx.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	query := `
		SELECT user_id, role, email, phone, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	if err := r.db.Get(&user, query, email); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) CreateCustomerUser(user *domain.User, customer *domain.CustomerProfile) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	return tx.Commit()
}

func (r *AuthRepository) CreateFactoryUser(user *domain.User, factory *domain.FactoryProfile) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	const factoryInsert = `
		INSERT INTO factory_profiles (user_id, factory_name, factory_type_id, tax_id)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.Exec(factoryInsert, user.UserID, factory.FactoryName, factory.FactoryTypeID, factory.TaxID); err != nil {
		return err
	}

	return tx.Commit()
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
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const updateUser = "UPDATE users SET password_hash = $1, updated_at = $2 WHERE user_id = $3"
	if _, err := tx.Exec(updateUser, passwordHash, now, userID); err != nil {
		return err
	}

	const markToken = "UPDATE password_reset_tokens SET used_at = $1 WHERE id = $2"
	if _, err := tx.Exec(markToken, now, tokenID); err != nil {
		return err
	}

	return tx.Commit()
}

func IsNotFoundError(err error) bool {
	return err == sql.ErrNoRows
}
