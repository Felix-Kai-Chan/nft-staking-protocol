package service

import (
	"context"
	"errors"
	"math/big"

	"staking-offchain/internal/contract"
	"staking-offchain/internal/dao"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type StakeService struct {
	stakeDAO     *dao.StakeDAO
	claimDAO     *dao.ClaimDAO
	contract     *contract.Contract
	transactOpts *bind.TransactOpts
}

func NewStakeService(
	stakeDAO *dao.StakeDAO,
	claimDAO *dao.ClaimDAO,
	contractInstance *contract.Contract,
	opts *bind.TransactOpts,
) *StakeService {
	return &StakeService{
		stakeDAO:     stakeDAO,
		claimDAO:     claimDAO,
		contract:     contractInstance,
		transactOpts: opts,
	}
}

func (s *StakeService) GetStakeInfo(ctx context.Context, address string) (*dao.Stake, error) {
	return s.stakeDAO.GetStake(ctx, address)
}

func (s *StakeService) GetClaims(ctx context.Context, address string) ([]dao.Claim, error) {
	return s.claimDAO.GetClaims(ctx, address)
}

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

func (s *StakeService) Claim(ctx context.Context) (string, error) {
	tx, err := s.contract.ClaimReward(s.transactOpts)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}
