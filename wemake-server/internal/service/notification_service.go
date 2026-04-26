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

func (s *NotificationService) ListPaginated(userID int64, page, limit int, unreadOnly bool) ([]domain.Notification, int64, int64, error) {
	return s.repo.ListPaginated(userID, page, limit, unreadOnly)
}

func (s *NotificationService) GetUnreadCount(userID int64) (int64, error) {
	return s.repo.GetUnreadCount(userID)
}

func (s *NotificationService) MarkAllRead(userID int64) (int64, error) {
	return s.repo.MarkAllRead(userID)
}

func (s *NotificationService) SoftDelete(notiID, userID int64) error {
	return s.repo.SoftDelete(notiID, userID)
}
