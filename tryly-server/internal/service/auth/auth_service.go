package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/dbutil"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	authrepo "github.com/yourusername/wemake/internal/repository/auth"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidRole         = errors.New("invalid role")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserInactive        = errors.New("user is inactive")
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
	ErrMissingRoleData     = errors.New("missing required fields for role")
	ErrNotCustomerAccount  = errors.New("account is not a customer account")
	ErrFactoryAlreadySetup  = errors.New("factory profile already exists for this account")
	ErrCustomerAlreadySetup = errors.New("customer profile already exists for this account")
	ErrMissingProfile       = errors.New("no profile exists for the requested role")
	ErrAlreadyActiveRole    = errors.New("this role is already active")
)

type AuthService struct {
	repo      *authrepo.AuthRepository
	jwtSecret string
}

type RegisterInput struct {
	Role           string
	Email          string
	Phone          string
	Password       string
	FirstName      string
	LastName       string
	FactoryName    string
	TaxID          string
	ProvinceID     *int64
	CategoryIDs    []int64
	SubCategoryIDs []int64
	// Cert fields for FT role
	CertID         int64
	DocumentURL    string
	CertNumber     string
	CertExpireDate string
	// Address fields — inserted as default address row
	AddressDetail string
	SubDistrictID int64
	DistrictID    int64
	ZipCode       string
}

type LoginResult struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func NewAuthService(repo *authrepo.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: jwtSecret}
}

func (s *AuthService) GetUserByID(userID int64) (*domain.User, error) {
	return s.repo.GetUserByID(userID)
}

// CheckEmailExists looks up whether an email is already registered.
// Returns the user (with Role populated) if found, or nil if not found.
func (s *AuthService) CheckEmailExists(email string) (*domain.User, error) {
	return s.repo.GetUserByEmail(domainutil.NormalizeLower(email))
}

func (s *AuthService) Register(input RegisterInput) (*LoginResult, error) {
	input.Email = domainutil.NormalizeLower(input.Email)
	input.Role = domainutil.NormalizeStatus(input.Role)

	if input.Role != domain.RoleCustomer && input.Role != domain.RoleFactory {
		return nil, ErrInvalidRole
	}

	if _, err := s.repo.GetUserByEmail(input.Email); err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !dbutil.IsNotFoundError(err) {
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
		if strings.TrimSpace(input.FactoryName) == "" {
			return nil, ErrMissingRoleData
		}
		factory := &domain.FactoryProfile{
			FactoryName:   strings.TrimSpace(input.FactoryName),
			TaxID:         strings.TrimSpace(input.TaxID),
			ProvinceID:    input.ProvinceID,
		}
		var addr *domain.Address
		if input.SubDistrictID > 0 || input.DistrictID > 0 || (input.ProvinceID != nil && *input.ProvinceID > 0) {
			pid := int64(0)
			if input.ProvinceID != nil {
				pid = *input.ProvinceID
			}
			addr = &domain.Address{
				AddressDetail: strings.TrimSpace(input.AddressDetail),
				SubDistrictID: input.SubDistrictID,
				DistrictID:    input.DistrictID,
				ProvinceID:    pid,
				ZipCode:       strings.TrimSpace(input.ZipCode),
				IsDefault:     true,
			}
		}
		if err := s.repo.CreateFactoryUser(user, factory, input.CategoryIDs, input.SubCategoryIDs, input.CertID, input.DocumentURL, input.CertNumber, input.CertExpireDate, addr); err != nil {
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

// UpgradeToFactory adds a factory profile to an existing user and switches to FT.
// Works for any user that doesn't already have a factory_profiles row.
func (s *AuthService) UpgradeToFactory(userID int64, input RegisterInput) (*LoginResult, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// Block if already has a factory profile
	hasFT, _ := s.repo.HasProfile(userID, domain.RoleFactory)
	if hasFT {
		return nil, ErrFactoryAlreadySetup
	}

	if strings.TrimSpace(input.FactoryName) == "" {
		return nil, ErrMissingRoleData
	}

	factory := &domain.FactoryProfile{
		FactoryName:   strings.TrimSpace(input.FactoryName),
		TaxID:         strings.TrimSpace(input.TaxID),
		ProvinceID:    input.ProvinceID,
	}
	var addr *domain.Address
	if input.SubDistrictID > 0 || input.DistrictID > 0 || (input.ProvinceID != nil && *input.ProvinceID > 0) {
		pid := int64(0)
		if input.ProvinceID != nil {
			pid = *input.ProvinceID
		}
		addr = &domain.Address{
			AddressDetail: strings.TrimSpace(input.AddressDetail),
			SubDistrictID: input.SubDistrictID,
			DistrictID:    input.DistrictID,
			ProvinceID:    pid,
			ZipCode:       strings.TrimSpace(input.ZipCode),
			IsDefault:     true,
		}
	}
	if err := s.repo.UpgradeToFactory(
		userID, factory,
		input.CategoryIDs, input.SubCategoryIDs,
		input.CertID, input.DocumentURL, input.CertNumber, input.CertExpireDate,
		addr,
	); err != nil {
		return nil, err
	}

	// Build updated user object with FT role for token generation
	user.Role = domain.RoleFactory
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return &LoginResult{Token: token, User: user}, nil
}

func (s *AuthService) HasFactoryProfile(userID int64) (bool, error) {
	return s.repo.HasProfile(userID, domain.RoleFactory)
}

func (s *AuthService) HasCustomerProfile(userID int64) (bool, error) {
	return s.repo.HasProfile(userID, domain.RoleCustomer)
}

func (s *AuthService) UpgradeToCustomer(userID int64, firstName, lastName string) (*LoginResult, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	hasCT, _ := s.repo.HasProfile(userID, domain.RoleCustomer)
	if hasCT {
		return nil, ErrCustomerAlreadySetup
	}

	if strings.TrimSpace(firstName) == "" || strings.TrimSpace(lastName) == "" {
		return nil, ErrMissingRoleData
	}

	if err := s.repo.UpgradeToCustomer(userID, strings.TrimSpace(firstName), strings.TrimSpace(lastName)); err != nil {
		return nil, err
	}

	user.Role = domain.RoleCustomer
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return &LoginResult{Token: token, User: user}, nil
}

// GetAvailableRoles returns all roles that a user has profiles for,
// plus their current active role.
func (s *AuthService) GetAvailableRoles(userID int64) ([]string, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	var roles []string
	hasCT, _ := s.repo.HasProfile(userID, domain.RoleCustomer)
	hasFT, _ := s.repo.HasProfile(userID, domain.RoleFactory)
	if hasCT {
		roles = append(roles, domain.RoleCustomer)
	}
	if hasFT {
		roles = append(roles, domain.RoleFactory)
	}
	// If neither profile found, at least return current role
	if len(roles) == 0 {
		roles = append(roles, strings.ToUpper(user.Role))
	}
	return roles, nil
}

// SwitchRole changes the active role for a dual-profile user.
// The target role must be CT or FT, different from the current role,
// and the user must have the matching profile row.
func (s *AuthService) SwitchRole(userID int64, targetRole string) (*LoginResult, error) {
	if targetRole != domain.RoleCustomer && targetRole != domain.RoleFactory {
		return nil, ErrInvalidRole
	}

	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if strings.ToUpper(user.Role) == targetRole {
		return nil, ErrAlreadyActiveRole
	}

	// Check the user actually has a profile for the target role
	hasProfile, err := s.repo.HasProfile(userID, targetRole)
	if err != nil {
		return nil, err
	}
	if !hasProfile {
		return nil, ErrMissingProfile
	}

	if err := s.repo.UpdateRole(userID, targetRole); err != nil {
		return nil, err
	}

	user.Role = targetRole
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
	actorRole = domainutil.NormalizeStatus(actorRole)
	if actorRole != domain.RoleSuperAdmin {
		return nil, ErrInvalidRole
	}

	input.Role = domainutil.NormalizeStatus(input.Role)
	if input.Role != domain.RoleAccountManager && input.Role != domain.RoleAdmin && input.Role != domain.RoleSuperAdmin {
		return nil, ErrInvalidRole
	}
	if input.Role == domain.RoleSuperAdmin {
		return nil, ErrInvalidRole
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		return nil, ErrMissingRoleData
	}

	input.Email = domainutil.NormalizeLower(input.Email)
	if _, err := s.repo.GetUserByEmail(input.Email); err == nil {
		return nil, ErrEmailAlreadyExists
	} else if !dbutil.IsNotFoundError(err) {
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
	normalizedEmail := domainutil.NormalizeLower(email)

	user, err := s.repo.GetUserByEmail(normalizedEmail)
	if err != nil {
		if dbutil.IsNotFoundError(err) {
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
	user, err := s.repo.GetUserByEmail(domainutil.NormalizeLower(email))
	if err != nil {
		if dbutil.IsNotFoundError(err) {
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
		if dbutil.IsNotFoundError(err) {
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
