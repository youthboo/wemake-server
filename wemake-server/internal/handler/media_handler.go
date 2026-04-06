package handler

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MediaHandler struct {
	publicBaseURL string
	cld           *cloudinary.Cloudinary
}

func NewMediaHandler(publicBaseURL string, cld *cloudinary.Cloudinary) *MediaHandler {
	return &MediaHandler{
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
		cld:           cld,
	}
}

func (h *MediaHandler) UploadFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required in form-data"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".jpg"
	}

	newFilename := uuid.New().String() + ext
	publicID := strings.TrimSuffix(newFilename, ext)

	if h.cld != nil {
		src, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read upload"})
		}
		defer src.Close()

		result, err := h.cld.Upload.Upload(context.Background(), src, uploader.UploadParams{
			Folder:       "wemake",
			PublicID:     publicID,
			ResourceType: "auto",
		})
		if err != nil {
			log.Printf("cloudinary upload: %v", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to upload to cloud storage"})
		}
		if result.SecureURL == "" {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "cloud storage returned empty URL"})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"url":       result.SecureURL,
			"file_name": result.PublicID,
			"size":      file.Size,
		})
	}

	saveDir := "./uploads"
	savePath := filepath.Join(saveDir, newFilename)

	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create uploads directory"})
	}

	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}

	baseURL := h.publicBaseURL
	if baseURL == "" {
		baseURL = c.BaseURL()
	}

	fileURL := fmt.Sprintf("%s/uploads/%s", baseURL, newFilename)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"url":       fileURL,
		"file_name": newFilename,
		"size":      file.Size,
	})
}
