package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type AdminUserHandler struct {
	authService *service.AuthService
	authRepo    *repository.AuthRepository
}

func NewAdminUserHandler(authService *service.AuthService, authRepo *repository.AuthRepository) *AdminUserHandler {
	return &AdminUserHandler{authService: authService, authRepo: authRepo}
}

func (h *AdminUserHandler) Create(c *fiber.Ctx) error {
	actorID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	actor, err := h.authService.GetUserByID(actorID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req struct {
		Email       string  `json:"email"`
		Password    string  `json:"password"`
		Role        string  `json:"role"`
		DisplayName string  `json:"display_name"`
		Department  *string `json:"department"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	item, err := h.authService.RegisterAdmin(service.RegisterAdminInput{
		Role:        strings.TrimSpace(req.Role),
		Email:       strings.TrimSpace(req.Email),
		Password:    req.Password,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Department:  req.Department,
		CreatedBy:   &actorID,
	}, actor.Role)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(item.User)
}

func (h *AdminUserHandler) List(c *fiber.Ctx) error {
	items, err := h.authRepo.ListAdminUsers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch admin users"})
	}
	return c.JSON(fiber.Map{"data": items})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizePageSize(size int) int {
	if size <= 0 {
		return 20
	}
	if size > 100 {
		return 100
	}
	return size
}
