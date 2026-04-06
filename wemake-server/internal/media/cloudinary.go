package media

import (
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/yourusername/wemake/internal/config"
)

// NewCloudinaryClient returns an upload-capable client when env is set, or (nil, nil) if disabled.
func NewCloudinaryClient(cfg *config.Config) (*cloudinary.Cloudinary, error) {
	if cfg == nil {
		return nil, nil
	}
	if u := strings.TrimSpace(cfg.CloudinaryURL); u != "" {
		return cloudinary.NewFromURL(u)
	}
	if cfg.CloudinaryCloudName != "" && cfg.CloudinaryAPIKey != "" && cfg.CloudinaryAPISecret != "" {
		return cloudinary.NewFromParams(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
	}
	return nil, nil
}
