package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type MessageHandler struct {
	service *service.MessageService
}

func NewMessageHandler(service *service.MessageService) *MessageHandler {
	return &MessageHandler{service: service}
}

func (h *MessageHandler) CreateMessage(c *fiber.Ctx) error {
	type reqBody struct {
		ReferenceType string  `json:"reference_type"`
		ReferenceID   string  `json:"reference_id"`
		ReceiverID    int64   `json:"receiver_id"`
		Content       string  `json:"content"`
		AttachmentURL string  `json:"attachment_url"`
		ConvID        *int64  `json:"conv_id"`
		MessageType   string  `json:"message_type"`
		QuoteData     *string `json:"quote_data"`
	}
	senderID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.ReceiverID <= 0 || strings.TrimSpace(req.ReferenceType) == "" || strings.TrimSpace(req.ReferenceID) == "" || strings.TrimSpace(req.Content) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reference_type, reference_id, receiver_id, content are required"})
	}
	item := &domain.Message{
		ReferenceType: req.ReferenceType,
		ReferenceID:   req.ReferenceID,
		SenderID:      senderID,
		ReceiverID:    req.ReceiverID,
		Content:       req.Content,
		AttachmentURL: req.AttachmentURL,
		ConvID:        req.ConvID,
		MessageType:   req.MessageType,
		QuoteData:     req.QuoteData,
		IsRead:        false,
	}
	if err := h.service.Create(item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create message"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *MessageHandler) ListMessages(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}

	convID := c.QueryInt("conv_id", 0)
	if convID > 0 {
		items, err := h.service.ListByConvID(int64(convID))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch messages by conv_id"})
		}
		return c.JSON(items)
	}

	referenceType := c.Query("reference_type")
	referenceID := c.Query("reference_id")
	if strings.TrimSpace(referenceType) == "" || strings.TrimSpace(referenceID) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reference_type and reference_id (or conv_id) are required"})
	}
	items, err := h.service.ListByReference(referenceType, referenceID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch messages"})
	}
	return c.JSON(items)
}

func (h *MessageHandler) ListThreads(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	items, err := h.service.ListThreads(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch threads"})
	}
	return c.JSON(items)
}
