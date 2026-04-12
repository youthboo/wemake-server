package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var ErrTopupAlreadyProcessed = errors.New("topup intent already processed or expired")

type TopupService struct {
	repo       *repository.TopupRepository
	walletRepo *repository.WalletRepository
}

func NewTopupService(repo *repository.TopupRepository, walletRepo *repository.WalletRepository) *TopupService {
	return &TopupService{repo: repo, walletRepo: walletRepo}
}

func (s *TopupService) CreateIntent(userID int64, amount float64) (*domain.TopupIntent, error) {
	walletID, err := s.walletRepo.GetWalletIDByUserID(userID)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(15 * time.Minute)
	// QR payload is a simulated PromptPay reference string; replace with real PromptPay API in production.
	qrPayload := fmt.Sprintf("PROMPTPAY|%d|%.2f|%s", *walletID, amount, expiresAt.Format("20060102150405"))
	intent := &domain.TopupIntent{
		WalletID:  *walletID,
		Amount:    amount,
		QRPayload: &qrPayload,
		Status:    "PE",
		ExpiresAt: &expiresAt,
	}
	if err := s.repo.Create(intent); err != nil {
		return nil, err
	}
	return intent, nil
}

func (s *TopupService) GetIntent(intentID string) (*domain.TopupIntent, error) {
	return s.repo.GetByID(intentID)
}

func (s *TopupService) ConfirmIntent(intentID string) (*domain.TopupIntent, error) {
	intent, err := s.repo.Confirm(intentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTopupAlreadyProcessed
		}
		return nil, err
	}
	if err := s.repo.AddFunds(intent.WalletID, intent.Amount); err != nil {
		return nil, err
	}
	return intent, nil
}
