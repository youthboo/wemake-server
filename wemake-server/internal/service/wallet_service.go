package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type WalletService struct {
	repo   *repository.WalletRepository
	txRepo *repository.TransactionRepository
}

func NewWalletService(repo *repository.WalletRepository, txRepo *repository.TransactionRepository) *WalletService {
	return &WalletService{repo: repo, txRepo: txRepo}
}

func (s *WalletService) GetByUserID(userID int64) (*domain.Wallet, error) {
	return s.repo.GetByUserID(userID)
}

func (s *WalletService) ListTransactionsByUserID(userID int64, orderID *int64, txType *string, status *string) ([]domain.Transaction, error) {
	walletID, err := s.repo.GetWalletIDByUserID(userID)
	if err != nil {
		return nil, err
	}
	filters := repository.TransactionFilters{
		WalletID: walletID,
		OrderID:  orderID,
		Type:     txType,
		Status:   status,
	}
	return s.txRepo.List(filters)
}
