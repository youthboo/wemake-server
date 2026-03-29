package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type NotificationService struct {
	repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) ListByUserID(userID int64) ([]domain.Notification, error) {
	return s.repo.ListByUserID(userID)
}

func (s *NotificationService) MarkAsRead(notiID, userID int64) error {
	return s.repo.MarkAsRead(notiID, userID)
}

func (s *NotificationService) Create(noti *domain.Notification) error {
	return s.repo.Create(noti)
}
