package notification

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/sse"
	notificationservice "github.com/yourusername/wemake/internal/service/notification"
)

type NotificationHandler struct {
	service *notificationservice.NotificationService
	hub     *sse.Hub
	tickets *sse.TicketStore
}

func NewNotificationHandler(service *notificationservice.NotificationService, hub *sse.Hub) *NotificationHandler {
	return &NotificationHandler{service: service, hub: hub, tickets: sse.NewTicketStore()}
}

// IssueStreamTicket mints a short-lived, single-use ticket for opening the SSE
// stream. The caller is authenticated normally (Authorization header / JWT) on
// this POST, so no secret ends up in a URL. The client then opens
// GET /notifications/stream?ticket=<value>.
func (h *NotificationHandler) IssueStreamTicket(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	ticket := h.tickets.Issue(userID)
	if ticket == "" {
		return helper.JSONError(c, fiber.StatusInternalServerError, "failed to issue stream ticket")
	}
	return c.JSON(fiber.Map{"ticket": ticket})
}

func filterTypes(filter string) []string {
	switch filter {
	case "rfq":
		return []string{"quote_received", "rfq_expired", "rfq_closed"}
	case "order":
		return []string{"order_confirmed", "order_status_changed", "production_updated", "order_completed", "payment_due"}
	default:
		return nil
	}
}

func (h *NotificationHandler) List(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	filter := c.Query("filter", "all")
	limit, offset := helper.LimitOffset(c, helper.DefaultPageSize)
	types := filterTypes(filter)

	items, total, unreadCount, err := h.service.ListWithFilter(userID, types, limit, offset)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch notifications")
	}
	return c.JSON(fiber.Map{
		"unread_count": unreadCount,
		"total":        total,
		"items":        items,
	})
}

func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	notiID, err := helper.RequireInt64Param(c, "noti_id")
	if err != nil {
		return err
	}
	unreadCount, err := h.service.MarkAsReadReturnCount(notiID, userID)
	if err != nil {
		return helper.JSONInternal(c, "failed to mark notification as read")
	}
	data, _ := json.Marshal(map[string]any{"noti_id": notiID, "unread_count": unreadCount})
	h.hub.Push(userID, "read", string(data))

	return c.JSON(fiber.Map{"ok": true, "unread_count": unreadCount})
}

func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	var body struct {
		Filter string `json:"filter"`
	}
	_ = c.BodyParser(&body)
	types := filterTypes(body.Filter)

	updatedCount, unreadCount, err := h.service.MarkAllReadWithFilter(userID, types)
	if err != nil {
		return helper.JSONInternal(c, "failed to mark all as read")
	}
	return c.JSON(fiber.Map{"ok": true, "updated_count": updatedCount, "unread_count": unreadCount})
}

func (h *NotificationHandler) GetUnreadCount(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	count, err := h.service.GetUnreadCount(userID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch unread count")
	}
	return c.JSON(fiber.Map{"count": count})
}

func (h *NotificationHandler) SoftDelete(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	notiID, err := helper.ParsePositiveInt64Param(c, "noti_id")
	if err != nil {
		return helper.BadRequestError(c, "invalid noti_id")
	}
	if err := h.service.SoftDelete(notiID, userID); err != nil {
		return helper.JSONInternal(c, "failed to delete notification")
	}
	return c.JSON(fiber.Map{"message": "notification deleted"})
}

type ssePayload struct {
	NotiID      int64  `json:"noti_id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	LinkTo      string `json:"link_to"`
	IsRead      bool   `json:"is_read"`
	CreatedAt   string `json:"created_at"`
	UnreadCount int64  `json:"unread_count"`
}

func (h *NotificationHandler) Stream(c *fiber.Ctx) error {
	// Prefer a single-use ticket (?ticket=) so a long-lived JWT never appears in
	// the URL (URLs leak into access/proxy logs and browser history). Fall back
	// to the existing header / ?token= auth for backward compatibility.
	userID, ok := h.tickets.Consume(c.Query("ticket"))
	if !ok {
		var err error
		userID, err = helper.RequireAuthenticatedUserID(c)
		if err != nil {
			return err
		}
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	lastNotiID := int64(0)
	if s := c.Get("Last-Event-ID"); s != "" {
		if id, parseErr := strconv.ParseInt(s, 10, 64); parseErr == nil {
			lastNotiID = id
		}
	}

	ch := h.hub.Subscribe(userID)

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer h.hub.Unsubscribe(userID, ch)

		count, _ := h.service.GetUnreadCount(userID)
		initData, _ := json.Marshal(map[string]any{"unread_count": count})
		fmt.Fprintf(w, "event: init\ndata: %s\n\n", initData)
		if err := w.Flush(); err != nil {
			return
		}

		pingTick := time.NewTicker(30 * time.Second)
		pollTick := time.NewTicker(5 * time.Second)
		defer pingTick.Stop()
		defer pollTick.Stop()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				line := "event: " + msg.Event + "\n"
				if msg.ID != "" {
					line += "id: " + msg.ID + "\n"
				}
				line += "data: " + msg.Data + "\n\n"
				if _, err := fmt.Fprint(w, line); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}

			case <-pollTick.C:
				rows, pollErr := h.service.PollNew(userID, lastNotiID)
				if pollErr != nil {
					continue
				}
				for _, row := range rows {
					if row.NotiID > lastNotiID {
						lastNotiID = row.NotiID
					}
					payload := ssePayload{
						NotiID:      row.NotiID,
						Type:        row.Type,
						Title:       row.Title,
						Message:     row.Message,
						LinkTo:      row.LinkTo,
						IsRead:      row.IsRead,
						CreatedAt:   row.CreatedAt,
						UnreadCount: row.UnreadCount,
					}
					data, _ := json.Marshal(payload)
					if _, err := fmt.Fprintf(w, "event: new_notification\nid: %d\ndata: %s\n\n", row.NotiID, data); err != nil {
						return
					}
				}
				if len(rows) > 0 {
					if err := w.Flush(); err != nil {
						return
					}
				}

			case <-pingTick.C:
				if _, err := fmt.Fprint(w, "event: ping\ndata: {}\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))
	return nil
}
