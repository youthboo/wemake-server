package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MediaHandler struct{}

func NewMediaHandler() *MediaHandler {
	return &MediaHandler{}
}

func (h *MediaHandler) UploadFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required in form-data"})
	}

	// Validate media type if needed, but requirements state:
	// type: "product" | "promotion" | "factory" | "rfq"
	// We could use this to organize folders, but saving in generic uploads is fine too.

	// Generate unique filename to avoid conflicts
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".jpg" // default if not provided
	}

	newFilename := uuid.New().String() + ext
	saveDir := "./uploads"
	savePath := filepath.Join(saveDir, newFilename)

	// Ensure uploads directory exists
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create uploads directory"})
	}

	// Save file locally
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}

	// Determine base URL, typically from env or req
	baseURL := c.BaseURL()

	fileURL := fmt.Sprintf("%s/uploads/%s", baseURL, newFilename)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"url":       fileURL,
		"file_name": newFilename,
		"size":      file.Size,
	})
}
