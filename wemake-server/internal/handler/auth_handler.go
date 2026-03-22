package handler

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type registerRequest struct {
	Role          string `json:"role"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Password      string `json:"password"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	FactoryName   string `json:"factory_name"`
	FactoryTypeID int64  `json:"factory_type_id"`
	TaxID         string `json:"tax_id"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.Role) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "role, email, and password are required"})
	}

	result, err := h.service.Register(service.RegisterInput{
		Role:          req.Role,
		Email:         req.Email,
		Phone:         req.Phone,
		Password:      req.Password,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		FactoryName:   req.FactoryName,
		FactoryTypeID: req.FactoryTypeID,
		TaxID:         req.TaxID,
	})
	if err != nil {
		switch err {
		case service.ErrEmailAlreadyExists:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		case service.ErrInvalidRole, service.ErrMissingRoleData:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		default:
			log.Printf("register failed: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "failed to register",
				"details": err.Error(),
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email and password are required"})
	}

	result, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		case service.ErrUserInactive:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to login"})
		}
	}

	return c.JSON(result)
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if strings.TrimSpace(req.Email) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
	}

	token, err := h.service.ForgotPassword(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to process forgot password"})
	}

	resp := fiber.Map{
		"message": "if the account exists, reset instructions have been generated",
	}
	if token != "" {
		resp["reset_token"] = token
	}

	return c.JSON(resp)
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if strings.TrimSpace(req.Token) == "" || strings.TrimSpace(req.NewPassword) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token and new_password are required"})
	}

	if err := h.service.ResetPassword(req.Token, req.NewPassword); err != nil {
		if err == service.ErrInvalidResetToken {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to reset password"})
	}

	return c.JSON(fiber.Map{"message": "password reset successful"})
}
