package service

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var ErrMaxRFQImages = errors.New("rfq image limit exceeded (max 4)")

type RFQService struct {
	repo *repository.RFQRepository
}

func NewRFQService(repo *repository.RFQRepository) *RFQService {
	return &RFQService{repo: repo}
}

func (s *RFQService) Create(rfq *domain.RFQ) error {
	now := time.Now()
	rfq.Title = strings.TrimSpace(rfq.Title)
	rfq.Details = strings.TrimSpace(rfq.Details)
	rfq.Status = "OP"
	rfq.CreatedAt = now
	rfq.UpdatedAt = now
	return s.repo.Create(rfq)
}

func (s *RFQService) ListByUserID(userID int64, status string) ([]domain.RFQ, error) {
	return s.repo.ListByUserID(userID, strings.TrimSpace(strings.ToUpper(status)))
}

func (s *RFQService) GetByID(userID, rfqID int64) (*domain.RFQ, []domain.RFQImage, error) {
	rfq, err := s.repo.GetByID(userID, rfqID)
	if err != nil {
		return nil, nil, err
	}
	images, err := s.repo.ListImages(rfqID)
	if err != nil {
		return nil, nil, err
	}
	return rfq, images, nil
}

func (s *RFQService) Cancel(userID, rfqID int64) error {
	return s.repo.Cancel(userID, rfqID)
}

func (s *RFQService) AddImage(rfqID int64, imageURL string) (*domain.RFQImage, error) {
	count, err := s.repo.CountImages(rfqID)
	if err != nil {
		return nil, err
	}
	if count >= 4 {
		return nil, ErrMaxRFQImages
	}
	image := &domain.RFQImage{
		ImageID:  "img-" + uuid.NewString(),
		RFQID:    rfqID,
		ImageURL: strings.TrimSpace(imageURL),
	}
	if err := s.repo.CreateImage(image); err != nil {
		return nil, err
	}
	return image, nil
}
