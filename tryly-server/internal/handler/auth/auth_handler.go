package auth

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/logger"
	authservice "github.com/yourusername/wemake/internal/service/auth"
)

type AuthHandler struct {
	service *authservice.AuthService
}

func NewAuthHandler(service *authservice.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"Role":     "role, email, and password are required",
		"Email":    "role, email, and password are required",
		"Password": "role, email, and password are required",
	}); err != nil {
		return err
	}

	result, err := h.service.Register(authservice.RegisterInput{
		Role:           req.Role,
		Email:          req.Email,
		Phone:          req.Phone,
		Password:       req.Password,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		FactoryName:    req.FactoryName,
		FactoryTypeID:  req.FactoryTypeID,
		TaxID:          req.TaxID,
		ProvinceID:     req.ProvinceID,
		CategoryIDs:    req.CategoryIDs,
		SubCategoryIDs: req.SubCategoryIDs,
		CertID:         req.CertID,
		DocumentURL:    req.DocumentURL,
		CertNumber:     req.CertNumber,
		CertExpireDate: req.CertExpireDate,
		AddressDetail:  req.AddressDetail,
		SubDistrictID:  req.SubDistrictID,
		DistrictID:     req.DistrictID,
		ZipCode:        req.ZipCode,
	})
	if err != nil {
		switch err {
		case authservice.ErrEmailAlreadyExists:
			return helper.WriteAPIError(c, helper.ConflictAPIError("EMAIL_EXISTS", "email already exists"))
		case authservice.ErrInvalidRole, authservice.ErrMissingRoleData:
			return helper.WriteAPIError(c, helper.BadRequestAPIError("INVALID_ROLE", err.Error()))
		default:
			logger.Error("user registration failed", "role", req.Role, "email", req.Email, "err", err)
			return helper.WriteAPIError(c, helper.InternalServerAPIError("REGISTER_FAILED", "failed to register"))
		}
	}

	c.Status(fiber.StatusCreated)
	return c.JSON(result)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"Email":    "email and password are required",
		"Password": "email and password are required",
	}); err != nil {
		return err
	}

	result, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		switch err {
		case authservice.ErrInvalidCredentials:
			return helper.WriteAPIError(c, helper.UnauthorizedAPIError("INVALID_CREDENTIALS", "invalid email or password"))
		case authservice.ErrUserInactive:
			return helper.WriteAPIError(c, helper.ForbiddenAPIError("USER_INACTIVE", "account is inactive"))
		default:
			log.Printf("[LOGIN] unexpected error for %s: %v", req.Email, err)
			return helper.WriteAPIError(c, helper.InternalServerAPIError("LOGIN_FAILED", "failed to login"))
		}
	}

	return c.JSON(result)
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req dto.ForgotPasswordRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"Email": "email is required",
	}); err != nil {
		return err
	}

	token, err := h.service.ForgotPassword(req.Email)
	if err != nil {
		return helper.WriteAPIError(c, helper.InternalServerAPIError("FORGOT_PASSWORD_FAILED", "failed to process forgot password"))
	}

	data := fiber.Map{}
	if token != "" {
		data["reset_token"] = token
	}

	return helper.WriteSuccess(c, "if the account exists, reset instructions have been generated", data)
}

// UpgradeToFactory upgrades the authenticated CT user to FT.
// POST /api/v1/auth/upgrade-to-factory  (requires RequireAuth)
func (h *AuthHandler) UpgradeToFactory(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return helper.WriteAPIError(c, helper.UnauthorizedAPIError("UNAUTHORIZED", "authentication required"))
	}

	var req dto.UpgradeToFactoryRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"FactoryName":   "factory_name is required",
		"FactoryTypeID": "factory_type_id is required",
	}); err != nil {
		return err
	}

	result, err := h.service.UpgradeToFactory(userID, authservice.RegisterInput{
		FactoryName:    req.FactoryName,
		FactoryTypeID:  req.FactoryTypeID,
		TaxID:          req.TaxID,
		ProvinceID:     req.ProvinceID,
		CategoryIDs:    req.CategoryIDs,
		SubCategoryIDs: req.SubCategoryIDs,
		CertID:         req.CertID,
		DocumentURL:    req.DocumentURL,
		CertNumber:     req.CertNumber,
		CertExpireDate: req.CertExpireDate,
		AddressDetail:  req.AddressDetail,
		SubDistrictID:  req.SubDistrictID,
		DistrictID:     req.DistrictID,
		ZipCode:        req.ZipCode,
	})
	if err != nil {
		switch err {
		case authservice.ErrNotCustomerAccount:
			return helper.WriteAPIError(c, helper.ForbiddenAPIError("NOT_CUSTOMER", "only customer accounts can be upgraded"))
		case authservice.ErrFactoryAlreadySetup:
			return helper.WriteAPIError(c, helper.ConflictAPIError("ALREADY_FACTORY", "factory profile already exists"))
		case authservice.ErrMissingRoleData:
			return helper.WriteAPIError(c, helper.BadRequestAPIError("MISSING_DATA", err.Error()))
		default:
			logger.Error("upgrade to factory failed", "user_id", userID, "err", err)
			return helper.WriteAPIError(c, helper.InternalServerAPIError("UPGRADE_FAILED", "failed to upgrade account"))
		}
	}

	return c.JSON(result)
}

// UpgradeToCustomer adds a customer profile to an existing FT user.
// POST /api/v1/auth/upgrade-to-customer  (requires RequireAuth)
func (h *AuthHandler) UpgradeToCustomer(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return helper.WriteAPIError(c, helper.UnauthorizedAPIError("UNAUTHORIZED", "authentication required"))
	}

	var req dto.UpgradeToCustomerRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"FirstName": "first_name is required",
		"LastName":  "last_name is required",
	}); err != nil {
		return err
	}

	result, err := h.service.UpgradeToCustomer(userID, req.FirstName, req.LastName)
	if err != nil {
		switch err {
		case authservice.ErrCustomerAlreadySetup:
			return helper.WriteAPIError(c, helper.ConflictAPIError("ALREADY_CUSTOMER", "customer profile already exists"))
		case authservice.ErrMissingRoleData:
			return helper.WriteAPIError(c, helper.BadRequestAPIError("MISSING_DATA", err.Error()))
		default:
			logger.Error("upgrade to customer failed", "user_id", userID, "err", err)
			return helper.WriteAPIError(c, helper.InternalServerAPIError("UPGRADE_FAILED", "failed to upgrade account"))
		}
	}

	return c.JSON(result)
}

// SwitchRole toggles the active role for a dual-profile user (CT↔FT).
// POST /api/v1/auth/switch-role  (requires RequireAuth)
func (h *AuthHandler) SwitchRole(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return helper.WriteAPIError(c, helper.UnauthorizedAPIError("UNAUTHORIZED", "authentication required"))
	}

	var req dto.SwitchRoleRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"Role": "role is required",
	}); err != nil {
		return err
	}

	result, err := h.service.SwitchRole(userID, strings.ToUpper(strings.TrimSpace(req.Role)))
	if err != nil {
		switch err {
		case authservice.ErrInvalidRole:
			return helper.WriteAPIError(c, helper.BadRequestAPIError("INVALID_ROLE", "role must be CT or FT"))
		case authservice.ErrMissingProfile:
			return helper.WriteAPIError(c, helper.ForbiddenAPIError("NO_PROFILE", "no profile exists for the requested role"))
		case authservice.ErrAlreadyActiveRole:
			return helper.WriteAPIError(c, helper.BadRequestAPIError("ALREADY_ACTIVE", "this role is already active"))
		default:
			logger.Error("switch role failed", "user_id", userID, "err", err)
			return helper.WriteAPIError(c, helper.InternalServerAPIError("SWITCH_FAILED", "failed to switch role"))
		}
	}

	return c.JSON(result)
}

// AvailableRoles returns the list of roles a user has profiles for.
// GET /api/v1/auth/available-roles  (requires RequireAuth)
func (h *AuthHandler) AvailableRoles(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return helper.WriteAPIError(c, helper.UnauthorizedAPIError("UNAUTHORIZED", "authentication required"))
	}

	roles, err := h.service.GetAvailableRoles(userID)
	if err != nil {
		logger.Error("available roles failed", "user_id", userID, "err", err)
		return helper.WriteAPIError(c, helper.InternalServerAPIError("ROLES_FAILED", "failed to get available roles"))
	}

	return c.JSON(fiber.Map{"roles": roles})
}

// CheckEmail godoc
// GET /api/v1/auth/email-check?email=xxx
// No auth required. Returns { exists: bool, role: string|null, has_factory: bool }
func (h *AuthHandler) CheckEmail(c *fiber.Ctx) error {
	email := strings.TrimSpace(strings.ToLower(c.Query("email")))
	if email == "" {
		return helper.WriteAPIError(c, helper.BadRequestAPIError("MISSING_EMAIL", "email is required"))
	}
	user, err := h.service.CheckEmailExists(email)
	if err != nil || user == nil {
		return c.JSON(fiber.Map{"exists": false, "role": nil, "has_factory": false, "has_customer": false})
	}
	hasFT, _ := h.service.HasFactoryProfile(user.UserID)
	hasCT, _ := h.service.HasCustomerProfile(user.UserID)
	return c.JSON(fiber.Map{"exists": true, "role": user.Role, "has_factory": hasFT, "has_customer": hasCT})
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req dto.ResetPasswordRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"Token":       "token and new_password are required",
		"NewPassword": "token and new_password are required",
	}); err != nil {
		return err
	}

	if err := h.service.ResetPassword(req.Token, req.NewPassword); err != nil {
		if err == authservice.ErrInvalidResetToken {
			return helper.WriteAPIError(c, helper.BadRequestAPIError("INVALID_RESET_TOKEN", "invalid or expired reset token"))
		}
		return helper.WriteAPIError(c, helper.InternalServerAPIError("RESET_PASSWORD_FAILED", "failed to reset password"))
	}

	return helper.WriteSuccess(c, "password reset successful", nil)
}
