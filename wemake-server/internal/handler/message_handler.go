package handler

import (
	"encoding/json"
	"errors"
	"strconv"
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

type createMessageRequest struct {
	ReferenceType *string `json:"reference_type"`
	ReferenceID   *int64  `json:"reference_id"`
	ReceiverID    *int64  `json:"receiver_id"`
	Content       *string `json:"content"`
	AttachmentURL *string `json:"attachment_url"`
	ConvID        *int64  `json:"conv_id"`
	MessageType   *string `json:"message_type"`
	QuoteData     *string `json:"quote_data"`
}

func parseCreateMessageRequest(body []byte) (*createMessageRequest, error) {
	var req createMessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (h *MessageHandler) CreateMessage(c *fiber.Ctx) error {
	senderID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	req, err := parseCreateMessageRequest(c.Body())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.ReceiverID == nil || *req.ReceiverID <= 0 || req.Content == nil || strings.TrimSpace(*req.Content) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "receiver_id and content are required"})
	}

	var referenceType string
	if req.ReferenceType != nil {
		referenceType = strings.TrimSpace(*req.ReferenceType)
	}

	var attachmentURL string
	if req.AttachmentURL != nil {
		attachmentURL = strings.TrimSpace(*req.AttachmentURL)
	}

	var messageType string
	if req.MessageType != nil {
		messageType = strings.TrimSpace(*req.MessageType)
	}

	item := &domain.Message{
		ReferenceType: referenceType,
		ReferenceID:   valueOrZero(req.ReferenceID),
		SenderID:      senderID,
		ReceiverID:    *req.ReceiverID,
		Content:       *req.Content,
		AttachmentURL: attachmentURL,
		ConvID:        req.ConvID,
		MessageType:   messageType,
		QuoteData:     req.QuoteData,
		IsRead:        false,
	}
	if err := h.service.Create(item); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidReferenceType):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reference_type must be one of RQ, OD, PD, PM, ID"})
		case errors.Is(err, service.ErrReferencePairRequired):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reference_type and reference_id must be provided together"})
		case errors.Is(err, service.ErrReferenceNotFound):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reference_id not found for reference_type=" + item.ReferenceType})
		case errors.Is(err, service.ErrSenderReceiverSame):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "receiver_id must differ from sender_id"})
		case errors.Is(err, service.ErrConversationNotAccessible):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "conv_id must be a conversation the sender belongs to"})
		case errors.Is(err, service.ErrConversationReceiverMismatch):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "receiver_id must match the other participant in conv_id"})
		case errors.Is(err, service.ErrInvalidMessageType):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message_type must be one of TX, QT, IM"})
		case errors.Is(err, service.ErrQuoteDataRequired):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "quote_data is required when message_type is QT"})
		}
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
	referenceIDRaw := c.Query("reference_id")
	if strings.TrimSpace(referenceType) == "" || strings.TrimSpace(referenceIDRaw) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reference_type and reference_id (or conv_id) are required"})
	}
	referenceID, err := strconv.ParseInt(strings.TrimSpace(referenceIDRaw), 10, 64)
	if err != nil || referenceID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reference_id must be a positive integer"})
	}
	items, err := h.service.ListByReference(referenceType, referenceID, userID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidReferenceType) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reference_type must be one of RQ, OD, PD, PM, ID"})
		}
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

func valueOrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
