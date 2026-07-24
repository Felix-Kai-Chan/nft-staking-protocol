package service

import (
	"context"
	"errors"
	"math/big"

	"staking-offchain/internal/contract"
	"staking-offchain/internal/indexer/repository"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type StakeService struct {
	repo         *repository.Repository
	contract     *contract.Contract
	transactOpts *bind.TransactOpts
}

func NewStakeService(
	repo *repository.Repository,
	contractInstance *contract.Contract,
	opts *bind.TransactOpts,
) *StakeService {
	return &StakeService{
		repo:         repo,
		contract:     contractInstance,
		transactOpts: opts,
	}
}

// GetStakeInfo 获取用户质押信息
func (s *StakeService) GetStakeInfo(ctx context.Context, address string) (*repository.Stake, error) {
	return s.repo.GetStake(ctx, address)
}

// GetClaims 获取用户领取历史
func (s *StakeService) GetClaims(ctx context.Context, address string) ([]repository.Claim, error) {
	return s.repo.GetClaims(ctx, address)
}

// Stake 质押
func (s *StakeService) Stake(ctx context.Context, amount string) (string, error) {
	amountBig, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return "", errors.New("invalid amount")
	}

	tx, err := s.contract.Stake(s.transactOpts, amountBig)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

// Withdraw 提现
func (s *StakeService) Withdraw(ctx context.Context, amount string) (string, error) {
	amountBig, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return "", errors.New("invalid amount")
	}

	tx, err := s.contract.Withdraw(s.transactOpts, amountBig)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

// Claim 领取奖励
func (s *StakeService) Claim(ctx context.Context) (string, error) {
	tx, err := s.contract.ClaimReward(s.transactOpts)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}
