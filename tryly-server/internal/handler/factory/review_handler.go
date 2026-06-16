package factory

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/helper"
	userrepo "github.com/yourusername/wemake/internal/repository/user"
)

type ReviewHandler struct {
	reviews *userrepo.ReviewRepository
}

func NewReviewHandler(reviews *userrepo.ReviewRepository) *ReviewHandler {
	return &ReviewHandler{reviews: reviews}
}

// ListMyReviews GET /api/v1/factories/me/reviews
func (h *ReviewHandler) ListMyReviews(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	reviews, err := h.reviews.ListByFactoryID(factoryID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch reviews")
	}
	summary, err := h.reviews.GetSummaryByFactoryID(factoryID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch review summary")
	}
	return c.JSON(fiber.Map{
		"reviews": reviews,
		"summary": summary,
		"total":   len(reviews),
	})
}

// ReplyToReview POST /api/v1/factories/me/reviews/:id/reply
func (h *ReviewHandler) ReplyToReview(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	reviewID, err := c.ParamsInt("id")
	if err != nil {
		return helper.BadRequestError(c, "invalid review id")
	}

	var body struct {
		Reply string `json:"reply"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.BadRequestError(c, "invalid body")
	}
	body.Reply = strings.TrimSpace(body.Reply)
	if body.Reply == "" {
		return helper.BadRequestError(c, "reply cannot be empty")
	}
	if len([]rune(body.Reply)) > 1000 {
		return helper.BadRequestError(c, "reply too long (max 1000 chars)")
	}

	updated, err := h.reviews.ReplyByFactory(int64(reviewID), factoryID, factoryID, body.Reply)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "review not found")
		}
		return helper.JSONInternal(c, "failed to save reply")
	}
	return c.JSON(updated)
}
