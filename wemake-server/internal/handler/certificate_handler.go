package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type CertificateHandler struct {
	service *service.CertificateService
}

func NewCertificateHandler(service *service.CertificateService) *CertificateHandler {
	return &CertificateHandler{service: service}
}

func (h *CertificateHandler) ListByFactory(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	items, err := h.service.ListByFactoryID(int64(factoryID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch certificates"})
	}
	return c.JSON(items)
}

func (h *CertificateHandler) Create(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Assuming a factory user can only upload their own certificates
	if int64(factoryID) != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	var req domain.FactoryCertificate
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	req.FactoryID = int64(factoryID)

	if err := h.service.Create(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to upload certificate"})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}
