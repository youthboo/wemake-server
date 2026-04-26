package service

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
	ErrInvalidResetToken  = errors.New("invalid or expired reset token")
	ErrMissingRoleData    = errors.New("missing required fields for role")
)

type AuthService struct {
	repo      *repository.AuthRepository
	jwtSecret string
}

type RegisterInput struct {
	Role          string
	Email         string
	Phone         string
	Password      string
	FirstName     string
	LastName      string
	FactoryName   string
	FactoryTypeID int64
	TaxID         string
	ProvinceID    *int64
}

type LoginResult struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func NewAuthService(repo *repository.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: jwtSecret}
}

func (s *AuthService) GetUserByID(userID int64) (*domain.User, error) {
	return s.repo.GetUserByID(userID)
}

func (s *AuthService) Register(input RegisterInput) (*LoginResult, error) {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Role = strings.TrimSpace(strings.ToUpper(input.Role))

	if input.Role != domain.RoleCustomer && input.Role != domain.RoleFactory {
		return nil, ErrInvalidRole
	}

	if _, err := s.repo.GetUserByEmail(input.Email); err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !repository.IsNotFoundError(err) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		Role:         input.Role,
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: string(hashedPassword),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	switch input.Role {
	case domain.RoleCustomer:
		if strings.TrimSpace(input.FirstName) == "" || strings.TrimSpace(input.LastName) == "" {
			return nil, ErrMissingRoleData
		}
		customer := &domain.CustomerProfile{
			FirstName: strings.TrimSpace(input.FirstName),
			LastName:  strings.TrimSpace(input.LastName),
		}
		if err := s.repo.CreateCustomerUser(user, customer); err != nil {
			return nil, err
		}
	case domain.RoleFactory:
		if strings.TrimSpace(input.FactoryName) == "" || input.FactoryTypeID <= 0 {
			return nil, ErrMissingRoleData
		}
		factory := &domain.FactoryProfile{
			FactoryName:   strings.TrimSpace(input.FactoryName),
			FactoryTypeID: input.FactoryTypeID,
			TaxID:         strings.TrimSpace(input.TaxID),
			ProvinceID:    input.ProvinceID,
		}
		if err := s.repo.CreateFactoryUser(user, factory); err != nil {
			return nil, err
		}
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return &LoginResult{Token: token, User: user}, nil
}

type RegisterAdminInput struct {
	Role        string
	Email       string
	Phone       string
	Password    string
	DisplayName string
	Department  *string
	CreatedBy   *int64
}

func (s *AuthService) RegisterAdmin(input RegisterAdminInput, actorRole string) (*LoginResult, error) {
	actorRole = strings.TrimSpace(strings.ToUpper(actorRole))
	if actorRole != domain.RoleSuperAdmin {
		return nil, ErrInvalidRole
	}

	input.Role = strings.TrimSpace(strings.ToUpper(input.Role))
	if input.Role != domain.RoleAccountManager && input.Role != domain.RoleAdmin && input.Role != domain.RoleSuperAdmin {
		return nil, ErrInvalidRole
	}
	if input.Role == domain.RoleSuperAdmin {
		return nil, ErrInvalidRole
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		return nil, ErrMissingRoleData
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if _, err := s.repo.GetUserByEmail(input.Email); err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !repository.IsNotFoundError(err) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		Role:         input.Role,
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: string(hashedPassword),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	profile := &domain.AdminProfile{
		DisplayName: strings.TrimSpace(input.DisplayName),
		Department:  input.Department,
		CreatedBy:   input.CreatedBy,
	}
	if err := s.repo.CreateAdminUser(user, profile); err != nil {
		return nil, err
	}
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return &LoginResult{Token: token, User: user}, nil
}

func (s *AuthService) Login(email, password string) (*LoginResult, error) {
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))

	user, err := s.repo.GetUserByEmail(normalizedEmail)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now()
	if err := s.repo.UpdateLoginTimestamp(user.UserID, now); err != nil {
		return nil, err
	}
	user.UpdatedAt = now

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return &LoginResult{Token: token, User: user}, nil
}

func (s *AuthService) ForgotPassword(email string) (string, error) {
	user, err := s.repo.GetUserByEmail(strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return "", nil
		}
		return "", err
	}

	now := time.Now()
	resetToken := &domain.PasswordResetToken{
		UserID:    user.UserID,
		Token:     uuid.NewString(),
		ExpiresAt: now.Add(15 * time.Minute),
		CreatedAt: now,
	}
	if err := s.repo.CreatePasswordResetToken(resetToken); err != nil {
		return "", err
	}

	return resetToken.Token, nil
}

func (s *AuthService) ResetPassword(token, newPassword string) error {
	resetToken, err := s.repo.GetValidPasswordResetToken(strings.TrimSpace(token))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return ErrInvalidResetToken
		}
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	return s.repo.ResetPassword(resetToken.UserID, resetToken.ID, string(hashedPassword), now)
}

func (s *AuthService) generateToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.UserID,
		"role":    user.Role,
		"email":   user.Email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
