package admin

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type AdminCatalogHandler struct {
	db *sqlx.DB
}

func NewAdminCatalogHandler(db *sqlx.DB) *AdminCatalogHandler {
	return &AdminCatalogHandler{db: db}
}

// PatchCategoryImg PATCH /api/admin/categories/:id/img
// Body: {"img": "<url>"}
func (h *AdminCatalogHandler) PatchCategoryImg(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category id"})
	}

	var body struct {
		Img string `json:"img"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	body.Img = strings.TrimSpace(body.Img)

	res, err := h.db.Exec(`UPDATE lbi_categories SET img = $1 WHERE category_id = $2`, body.Img, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update category image"})
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "category not found"})
	}
	return c.JSON(fiber.Map{"ok": true, "category_id": id, "img": body.Img})
}
