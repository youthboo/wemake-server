package service

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrProfileInvalidInput = errors.New("PROFILE_INVALID_INPUT")
	ErrProfileUnauthorized = errors.New("PROFILE_UNAUTHORIZED")
)

type ProfileService struct {
	profiles *repository.ProfileRepository
	auth     *repository.AuthRepository
}

func NewProfileService(profiles *repository.ProfileRepository, auth *repository.AuthRepository) *ProfileService {
	return &ProfileService{profiles: profiles, auth: auth}
}

func (s *ProfileService) GetProfile(userID int64) (*domain.ProfileResponse, error) {
	return s.profiles.GetProfile(userID)
}

func (s *ProfileService) UpdateCustomerProfile(userID int64, phone string, bio *string, customer *domain.CustomerProfile) (*domain.ProfileResponse, error) {
	if strings.TrimSpace(customer.FirstName) == "" || strings.TrimSpace(customer.LastName) == "" {
		return nil, ErrProfileInvalidInput
	}
	if customer.PostalCode != nil && !regexp.MustCompile(`^\d{5}$`).MatchString(strings.TrimSpace(*customer.PostalCode)) {
		return nil, ErrProfileInvalidInput
	}
	user := &domain.User{Phone: strings.TrimSpace(phone), Bio: trimStringPtr(bio)}
	customer.FirstName = strings.TrimSpace(customer.FirstName)
	customer.LastName = strings.TrimSpace(customer.LastName)
	customer.AddressLine1 = trimStringPtr(customer.AddressLine1)
	customer.SubDistrict = trimStringPtr(customer.SubDistrict)
	customer.District = trimStringPtr(customer.District)
	customer.Province = trimStringPtr(customer.Province)
	customer.PostalCode = trimStringPtr(customer.PostalCode)
	if err := s.profiles.UpdateCustomerProfile(userID, user, customer); err != nil {
		return nil, err
	}
	return s.profiles.GetProfile(userID)
}

func (s *ProfileService) UpdateFactoryProfile(userID int64, phone string, bio *string, factory *domain.FactoryProfile) (*domain.ProfileResponse, error) {
	user := &domain.User{Phone: strings.TrimSpace(phone), Bio: trimStringPtr(bio)}
	factory.Description = trimStringPtr(factory.Description)
	factory.Specialization = trimStringPtr(factory.Specialization)
	factory.LeadTimeDesc = trimStringPtr(factory.LeadTimeDesc)
	factory.PriceRange = trimStringPtr(factory.PriceRange)
	if err := s.profiles.UpdateFactoryProfile(userID, user, factory); err != nil {
		return nil, err
	}
	return s.profiles.GetProfile(userID)
}

func (s *ProfileService) ChangePassword(userID int64, currentPassword, newPassword, confirmPassword string) error {
	user, err := s.auth.GetUserByID(userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return ErrProfileUnauthorized
	}
	if newPassword != confirmPassword || len(newPassword) < 8 || !regexp.MustCompile(`\d`).MatchString(newPassword) || newPassword == currentPassword {
		return ErrProfileInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.auth.UpdatePassword(userID, string(hash), time.Now())
}

func (s *ProfileService) UpdateAvatar(userID int64, avatarURL string) (*domain.ProfileResponse, error) {
	if err := s.profiles.UpdateAvatar(userID, avatarURL); err != nil {
		return nil, err
	}
	return s.profiles.GetProfile(userID)
}

func (s *ProfileService) GetSummary(userID int64, role string) (*domain.ProfileSummary, error) {
	return s.profiles.GetSummary(userID, role)
}

func (s *ProfileService) ListTransactions(userID int64, page, limit int, txType, status string) ([]domain.TransactionListItem, int64, float64, float64, error) {
	return s.profiles.ListTransactions(userID, page, limit, txType, status)
}

func (s *ProfileService) ListMyReviews(userID int64, page, limit int) ([]domain.UserReviewListItem, int64, error) {
	return s.profiles.ListMyReviews(userID, page, limit)
}

func (s *ProfileService) ListReceivedReviews(userID int64, role string, page, limit int) ([]domain.UserReviewListItem, int64, error) {
	if role != domain.RoleFactory {
		return nil, 0, ErrProfileUnauthorized
	}
	return s.profiles.ListReceivedReviews(userID, page, limit)
}

func (s *ProfileService) GetNotificationPreference(userID int64) (*domain.NotificationPreference, error) {
	return s.profiles.GetNotificationPreference(userID)
}

func (s *ProfileService) UpdateNotificationPreference(userID int64, item *domain.NotificationPreference) (*domain.NotificationPreference, error) {
	item.UserID = userID
	if err := s.profiles.UpsertNotificationPreference(item); err != nil {
		return nil, err
	}
	return s.profiles.GetNotificationPreference(userID)
}

func trimStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}
