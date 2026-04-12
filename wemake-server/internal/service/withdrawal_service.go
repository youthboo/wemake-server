package service

import (
	"errors"
	"strings"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var ErrInsufficientFunds = errors.New("insufficient wallet funds for withdrawal")
var ErrInvalidWithdrawalStatus = errors.New("status must be AP, RJ, or CP")

type WithdrawalService struct {
	repo       *repository.WithdrawalRepository
	walletRepo *repository.WalletRepository
}

func NewWithdrawalService(repo *repository.WithdrawalRepository, walletRepo *repository.WalletRepository) *WithdrawalService {
	return &WithdrawalService{repo: repo, walletRepo: walletRepo}
}

func (s *WithdrawalService) Create(factoryID int64, amount float64, bankAccountNo, bankName, accountName string) (*domain.WithdrawalRequest, error) {
	walletID, err := s.walletRepo.GetWalletIDByUserID(factoryID)
	if err != nil {
		return nil, err
	}
	wallet, err := s.walletRepo.GetByUserID(factoryID)
	if err != nil {
		return nil, err
	}
	if wallet.GoodFund < amount {
		return nil, ErrInsufficientFunds
	}
	w := &domain.WithdrawalRequest{
		WalletID:      *walletID,
		FactoryID:     factoryID,
		Amount:        amount,
		BankAccountNo: bankAccountNo,
		BankName:      bankName,
		AccountName:   accountName,
	}
	if err := s.repo.Create(w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WithdrawalService) ListByFactoryID(factoryID int64) ([]domain.WithdrawalRequest, error) {
	return s.repo.ListByFactoryID(factoryID)
}

func (s *WithdrawalService) UpdateStatus(requestID int64, status string, note *string) error {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "AP" && status != "RJ" && status != "CP" {
		return ErrInvalidWithdrawalStatus
	}
	return s.repo.UpdateStatus(requestID, status, note)
}
