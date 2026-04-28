package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type ConversationHandler struct {
	service *service.ConversationService
}

func NewConversationHandler(service *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{service: service}
}

func (h *ConversationHandler) List(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	items, err := h.service.ListByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch conversations"})
	}
	return c.JSON(items)
}

func (h *ConversationHandler) Get(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	convID, err := c.ParamsInt("conv_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid conv_id"})
	}
	item, err := h.service.GetByID(int64(convID), userID)
	if err != nil {
		if errors.Is(err, service.ErrConversationForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "conversation not found"})
	}
	return c.JSON(item)
}

func (h *ConversationHandler) Create(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req domain.Conversation
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	if req.CustomerID <= 0 || req.FactoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id and factory_id are required"})
	}
	// Allow both customer (CT) and factory (FT) to initiate a conversation room.
	// The caller must be one of the two parties — this is the security boundary.
	if req.CustomerID != userID && req.FactoryID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	if err := h.service.Create(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create conversation"})
	}
	item, err := h.service.GetByID(req.ConvID, userID)
	if err != nil {
		return c.Status(fiber.StatusCreated).JSON(req)
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *ConversationHandler) InquireShowcase(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid showcase_id"})
	}
	if role := getOptionalRoleFromContext(c); role != "" && role != domain.RoleCustomer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "buyer role required"})
	}
	item, err := h.service.CreateFromShowcase(showcaseID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create conversation"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"conv_id": item.ConvID})
}

func (h *ConversationHandler) MarkAsRead(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	convID, err := c.ParamsInt("conv_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid conv_id"})
	}
	if err := h.service.MarkAsRead(int64(convID), userID); err != nil {
		if errors.Is(err, service.ErrConversationNotFoundOrForbidden) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "conversation not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark conversation as read"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ConversationHandler) ShareRFQ(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if role := getOptionalRoleFromContext(c); role != "" && role != domain.RoleCustomer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "buyer role required"})
	}
	convID, err := c.ParamsInt("conv_id")
	if err != nil || convID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid conv_id"})
	}
	var req struct {
		RFQID int64 `json:"rfq_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	if req.RFQID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "rfq_id is required"})
	}
	msg, rfq, err := h.service.ShareRFQ(int64(convID), userID, req.RFQID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShareRFQForbidden):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		case errors.Is(err, service.ErrShareRFQClosed):
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "rfq cannot be shared"})
		case errors.Is(err, service.ErrShareRFQInvalid):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rfq or conversation not found"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to share rfq"})
		}
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": msg,
		"rfq":     rfq,
	})
}
