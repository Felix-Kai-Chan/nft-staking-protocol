package indexer

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"time"

	"staking-offchain/internal/indexer/repository"
	"staking-offchain/internal/infra/eth"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gorm.io/gorm"
)

// 事件签名常量
const (
	StakedEventSig    = "0x9e71bc8eea02a63969f509818f2dafb9254532904319f9dbda79b67bd34a5f3d"
	WithdrawnEventSig = "0x7084f5476618d8e60b11ef0d7d3f06914655adb8793e28ff7f018d4c76d505d5"
	ClaimedEventSig   = "0xd8138f8a3f377c5259ca548e70e4c2de94f129f5a11036a15b69513cba2b426a"
)

type Listener struct {
	ethClient    *eth.Client
	repo         *repository.Repository
	contractAddr common.Address
	chainID      int64
}

func NewListener(ethClient *eth.Client, repo *repository.Repository, contractAddr string, chainID int64) *Listener {
	return &Listener{
		ethClient:    ethClient,
		repo:         repo,
		contractAddr: common.HexToAddress(contractAddr),
		chainID:      chainID,
	}
}

// Start 开始监听链上事件
func (l *Listener) Start(ctx context.Context) {
	slog.Info("Indexer started listening for events", "mode", "polling")

	client := l.ethClient.EthClient

	// 1. 获取上次同步的区块号，如果没有则从创世块(0)开始
	startBlock, err := l.getStartingBlock(ctx)
	if err != nil {
		slog.Error("Failed to get starting block, exiting", "error", err)
		return
	}
	lastBlock := startBlock
	slog.Info("Starting to sync from block", "block", lastBlock)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Indexer stopped")
			return
		case <-ticker.C:
			currentHeader, err := client.HeaderByNumber(ctx, nil)
			if err != nil {
				slog.Warn("Failed to get latest header", "error", err)
				continue
			}

			currentBlock := currentHeader.Number.Uint64()
			// 如果链上没有新区块，就跳过本次循环
			if currentBlock <= lastBlock {
				continue
			}

			// 2. 分批处理区块，防止单次请求数据量过大
			toBlock := currentBlock
			if toBlock-lastBlock > 100 {
				toBlock = lastBlock + 100
			}

			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(lastBlock + 1)), // 从上次结束的下一个区块开始
				ToBlock:   big.NewInt(int64(toBlock)),
				Addresses: []common.Address{l.contractAddr},
			}

			logs, err := client.FilterLogs(ctx, query)
			if err != nil {
				slog.Warn("Failed to filter logs", "error", err)
				continue
			}

			slog.Debug("Processing logs", "from", lastBlock+1, "to", toBlock, "count", len(logs))
			for _, vLog := range logs {
				l.handleLog(ctx, vLog)
			}

			// 3. 成功处理完一批区块后，更新数据库中的同步状态
			if err := l.updateSyncStatus(ctx, toBlock); err != nil {
				slog.Error("Failed to update sync status, will retry next round", "error", err)
				continue
			}

			lastBlock = toBlock
			slog.Debug("Sync status updated", "last_block", lastBlock)
		}
	}
}

// getStartingBlock 从数据库获取上次同步的区块号
func (l *Listener) getStartingBlock(ctx context.Context) (uint64, error) {
	var status repository.SyncStatus
	err := l.repo.GetDB().WithContext(ctx).Where("id = ?", 1).First(&status).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	return status.LastBlock, nil
}

// updateSyncStatus 更新数据库中的同步状态
func (l *Listener) updateSyncStatus(ctx context.Context, blockNumber uint64) error {
	status := repository.SyncStatus{
		ID:        1,
		LastBlock: blockNumber,
		UpdatedAt: time.Now().Unix(),
	}
	return l.repo.GetDB().WithContext(ctx).Save(&status).Error
}

// handleLog 分发事件
func (l *Listener) handleLog(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) == 0 {
		return
	}
	eventSig := vLog.Topics[0].Hex()
	switch eventSig {
	case StakedEventSig:
		l.handleStaked(ctx, vLog)
	case WithdrawnEventSig:
		l.handleWithdrawn(ctx, vLog)
	case ClaimedEventSig:
		l.handleClaimed(ctx, vLog)
	default:
	}
}

// handleStaked 处理 Staked 事件
func (l *Listener) handleStaked(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 3 {
		slog.Warn("Invalid Staked event topics length", "topics", len(vLog.Topics), "expected", 3)
		return
	}
	user := common.BytesToAddress(vLog.Topics[1].Bytes())
	amount := new(big.Int).SetBytes(vLog.Topics[2].Bytes())
	slog.Info("Staked event received", "user", user.Hex(), "amount", amount.String())

	stake := &repository.Stake{
		UserAddress: user.Hex(),
		Amount:      amount.String(),
		StakedAt:    uint64(time.Now().Unix()),
		LastUpdated: uint64(time.Now().Unix()),
	}
	if err := l.repo.SaveStake(ctx, stake); err != nil {
		slog.Error("Failed to save stake", "error", err)
	}
}

// handleWithdrawn 处理 Withdrawn 事件 - 更新金额而不是删除
func (l *Listener) handleWithdrawn(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 3 {
		slog.Warn("Invalid Withdrawn event topics length", "topics", len(vLog.Topics), "expected", 3)
		return
	}
	user := common.BytesToAddress(vLog.Topics[1].Bytes())
	amount := new(big.Int).SetBytes(vLog.Topics[2].Bytes())
	slog.Info("Withdrawn event received", "user", user.Hex(), "amount", amount.String())

	// 获取当前质押记录
	stake, err := l.repo.GetStake(ctx, user.Hex())
	if err != nil {
		slog.Error("Failed to get stake for withdraw", "error", err)
		return
	}

	// 计算剩余金额
	currentAmount, ok := new(big.Int).SetString(stake.Amount, 10)
	if !ok {
		slog.Error("Failed to parse current amount", "amount", stake.Amount)
		return
	}
	newAmount := new(big.Int).Sub(currentAmount, amount)
	if newAmount.Sign() < 0 {
		slog.Warn("Withdraw amount exceeds staked amount, setting to 0", "user", user.Hex())
		newAmount = big.NewInt(0)
	}
	stake.Amount = newAmount.String()
	stake.LastUpdated = uint64(time.Now().Unix())

	if err := l.repo.UpdateStake(ctx, stake); err != nil {
		slog.Error("Failed to update stake after withdraw", "error", err)
	}
}

// handleClaimed 处理 Claimed 事件
func (l *Listener) handleClaimed(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 3 {
		slog.Warn("Invalid Claimed event topics length", "topics", len(vLog.Topics), "expected", 3)
		return
	}
	user := common.BytesToAddress(vLog.Topics[1].Bytes())
	reward := new(big.Int).SetBytes(vLog.Topics[2].Bytes())
	slog.Info("Claimed event received", "user", user.Hex(), "reward", reward.String())

	if err := l.repo.SaveClaim(ctx, user.Hex(), reward.String()); err != nil {
		slog.Error("Failed to save claim", "error", err)
	}
}
