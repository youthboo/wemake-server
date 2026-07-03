package media

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/logger"
)

type UploadOptions struct {
	FieldName             string
	FileName              string
	FileNamePrefix        string
	DefaultExt            string
	Folder                string
	SaveDir               string
	PublicBaseURL         string
	RequiredMessage       string
	MaxSizeMessage        string
	CloudUploadMessage    string
	CloudUploadLogMessage string
	CloudEmptyURLMessage  string
	MaxSize               int64
	Cloudinary            *cloudinary.Cloudinary
	CloudinaryLogFields   []interface{}
}

type UploadResult struct {
	URL      string
	FileName string
	PublicID string
	Size     int64
}

type UploadError struct {
	Status  int
	Message string
	Err     error
}

func (e *UploadError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func SaveUploadedFile(c *fiber.Ctx, opts UploadOptions) (*UploadResult, error) {
	fieldName := opts.FieldName
	if fieldName == "" {
		fieldName = "file"
	}
	file, err := c.FormFile(fieldName)
	if err != nil {
		return nil, uploadError(fiber.StatusBadRequest, defaultString(opts.RequiredMessage, "file is required"), err)
	}
	return saveFileHeader(c, file, opts)
}

// SaveUploadedFiles uploads every file posted under opts.FieldName (multiple
// files sharing the same form field name), in order. Fails fast on the first error.
func SaveUploadedFiles(c *fiber.Ctx, opts UploadOptions) ([]*UploadResult, error) {
	fieldName := opts.FieldName
	if fieldName == "" {
		fieldName = "file"
	}
	form, err := c.MultipartForm()
	if err != nil {
		return nil, uploadError(fiber.StatusBadRequest, defaultString(opts.RequiredMessage, "file is required"), err)
	}
	files := form.File[fieldName]
	if len(files) == 0 {
		return nil, uploadError(fiber.StatusBadRequest, defaultString(opts.RequiredMessage, "file is required"), nil)
	}
	results := make([]*UploadResult, 0, len(files))
	for i, fh := range files {
		fileOpts := opts
		if opts.FileNamePrefix != "" {
			fileOpts.FileNamePrefix = fmt.Sprintf("%s-%d", opts.FileNamePrefix, i)
		}
		result, err := saveFileHeader(c, fh, fileOpts)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func saveFileHeader(c *fiber.Ctx, file *multipart.FileHeader, opts UploadOptions) (*UploadResult, error) {
	if opts.MaxSize > 0 && file.Size > opts.MaxSize {
		return nil, uploadError(fiber.StatusBadRequest, defaultString(opts.MaxSizeMessage, "file exceeds maximum size"), nil)
	}

	defaultExt := defaultString(opts.DefaultExt, ".jpg")
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = defaultExt
	}
	fileName := opts.FileName
	if fileName == "" && opts.FileNamePrefix != "" {
		fileName = opts.FileNamePrefix + ext
	}
	if fileName == "" {
		fileName = strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename)) + ext
	}
	publicID := strings.TrimSuffix(fileName, ext)

	if opts.Cloudinary != nil {
		src, err := file.Open()
		if err != nil {
			return nil, uploadError(fiber.StatusInternalServerError, "failed to read upload", err)
		}
		defer src.Close()

		result, err := opts.Cloudinary.Upload.Upload(context.Background(), src, uploader.UploadParams{
			Folder:       opts.Folder,
			PublicID:     publicID,
			ResourceType: "auto",
		})
		if err != nil {
			logFields := append([]interface{}{"public_id", publicID, "err", err}, opts.CloudinaryLogFields...)
			logger.Error(defaultString(opts.CloudUploadLogMessage, "cloudinary upload failed"), logFields...)
			return nil, uploadError(fiber.StatusBadGateway, defaultString(opts.CloudUploadMessage, "failed to upload to cloud storage"), err)
		}
		if result.SecureURL == "" {
			return nil, uploadError(fiber.StatusBadGateway, defaultString(opts.CloudEmptyURLMessage, "cloud storage returned empty URL"), nil)
		}
		return &UploadResult{URL: result.SecureURL, FileName: result.PublicID, PublicID: result.PublicID, Size: file.Size}, nil
	}

	saveDir := defaultString(opts.SaveDir, "./uploads")
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, uploadError(fiber.StatusInternalServerError, "failed to create uploads directory", err)
	}
	if err := c.SaveFile(file, filepath.Join(saveDir, fileName)); err != nil {
		return nil, uploadError(fiber.StatusInternalServerError, "failed to save file", err)
	}

	baseURL := strings.TrimRight(opts.PublicBaseURL, "/")
	if baseURL == "" {
		baseURL = c.BaseURL()
	}
	return &UploadResult{URL: fmt.Sprintf("%s/uploads/%s", baseURL, fileName), FileName: fileName, PublicID: publicID, Size: file.Size}, nil
}

func uploadError(status int, message string, err error) *UploadError {
	return &UploadError{Status: status, Message: message, Err: err}
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
