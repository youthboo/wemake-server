package wallet

import (
	"errors"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
)

var ErrInsufficientFunds = errors.New("insufficient wallet funds for withdrawal")
var ErrInvalidWithdrawalStatus = errors.New("status must be AP, RJ, or CP")
var ErrSlipRequiredForComplete = errors.New("slip_url is required when completing a withdrawal")

type WithdrawalService struct {
	repo       *walletrepo.WithdrawalRepository
	walletRepo *walletrepo.WalletRepository
}

func NewWithdrawalService(repo *walletrepo.WithdrawalRepository, walletRepo *walletrepo.WalletRepository) *WithdrawalService {
	return &WithdrawalService{repo: repo, walletRepo: walletRepo}
}

func (s *WithdrawalService) Create(factoryID int64, amount float64, bankAccountNo, bankName, accountName string) (*domain.WithdrawalRequest, error) {
	walletID, err := s.walletRepo.GetWalletIDByUserID(factoryID)
	if err != nil {
		return nil, err
	}
	w := &domain.WithdrawalRequest{
		WalletID:      *walletID,
		FactoryID:     factoryID,
		Amount:        helper.MoneyDecimal(amount),
		BankAccountNo: bankAccountNo,
		BankName:      bankName,
		AccountName:   accountName,
	}
	// Repo holds the funds (good_fund → pending_fund) atomically with a
	// balance guard in the UPDATE, so concurrent requests cannot overdraw.
	if err := s.repo.Create(w); err != nil {
		if errors.Is(err, walletrepo.ErrWithdrawalHoldFailed) {
			return nil, ErrInsufficientFunds
		}
		return nil, err
	}
	return w, nil
}

func (s *WithdrawalService) ListByFactoryID(factoryID int64) ([]domain.WithdrawalRequest, error) {
	return s.repo.ListByFactoryID(factoryID)
}

func (s *WithdrawalService) UpdateStatus(requestID int64, status string, note *string, slipURL *string, processedBy *int64) error {
	status = domainutil.NormalizeStatus(status)
	if !domainutil.StatusIn(status, "AP", "RJ", "CP") {
		return ErrInvalidWithdrawalStatus
	}
	// เมื่อโอนเสร็จ (CP) superadmin ต้องแนบสลิปโอนเงินเสมอ
	if status == "CP" && (slipURL == nil || *slipURL == "") {
		return ErrSlipRequiredForComplete
	}
	return s.repo.UpdateStatus(requestID, status, note, slipURL, processedBy)
}
