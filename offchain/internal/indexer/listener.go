package indexer

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"staking-offchain/internal/dao"
	"staking-offchain/internal/infra/eth"
	"staking-offchain/internal/mq"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	StakedEventSig    = "0x9e71bc8eea02a63969f509818f2dafb9254532904319f9dbda79b67bd34a5f3d"
	WithdrawnEventSig = "0x7084f5476618d8e60b11ef0d7d3f06914655adb8793e28ff7f018d4c76d505d5"
	ClaimedEventSig   = "0xd8138f8a3f377c5259ca548e70e4c2de94f129f5a11036a15b69513cba2b426a"
)

type Listener struct {
	ethClient    *eth.Client
	producer     *mq.Producer
	cursorDAO    *dao.CursorDAO
	contractAddr common.Address
	chainID      int64
}

func NewListener(ethClient *eth.Client, producer *mq.Producer, cursorDAO *dao.CursorDAO, contractAddr string, chainID int64) *Listener {
	return &Listener{
		ethClient:    ethClient,
		producer:     producer,
		cursorDAO:    cursorDAO,
		contractAddr: common.HexToAddress(contractAddr),
		chainID:      chainID,
	}
}

func (l *Listener) Start(ctx context.Context) {
	slog.Info("Indexer started listening for events", "mode", "polling")

	client := l.ethClient.EthClient

	lastBlock, err := l.cursorDAO.GetLastBlock(ctx)
	if err != nil {
		slog.Error("Failed to get starting block", "error", err)
		return
	}
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
			safeBlock := currentBlock

			if safeBlock <= lastBlock {
				continue
			}

			toBlock := safeBlock
			if toBlock-lastBlock > 100 {
				toBlock = lastBlock + 100
			}

			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(lastBlock + 1)),
				ToBlock:   big.NewInt(int64(toBlock)),
				Addresses: []common.Address{l.contractAddr},
			}

			logs, err := client.FilterLogs(ctx, query)
			if err != nil {
				slog.Warn("Failed to filter logs", "error", err)
				continue
			}

			slog.Info("Processing logs", "from", lastBlock+1, "to", toBlock, "count", len(logs))

			for _, vLog := range logs {
				l.handleLog(ctx, vLog)
			}

			if err := l.cursorDAO.UpdateLastBlock(ctx, toBlock); err != nil {
				slog.Error("Failed to update cursor", "error", err)
				continue
			}

			lastBlock = toBlock
			slog.Info("Cursor updated", "last_block", lastBlock)
		}
	}
}

func (l *Listener) handleLog(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) == 0 {
		return
	}
	eventSig := vLog.Topics[0].Hex()
	switch eventSig {
	case StakedEventSig:
		l.publishStaked(ctx, vLog)
	case WithdrawnEventSig:
		l.publishWithdrawn(ctx, vLog)
	case ClaimedEventSig:
		l.publishClaimed(ctx, vLog)
	}
}

func (l *Listener) publishStaked(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 3 {
		return
	}
	user := common.BytesToAddress(vLog.Topics[1].Bytes())
	amount := new(big.Int).SetBytes(vLog.Topics[2].Bytes())

	slog.Info("Staked event", "user", user.Hex(), "amount", amount.String())

	msg := &mq.EventMessage{
		TxHash:      vLog.TxHash.Hex(),
		LogIndex:    vLog.Index,
		BlockNumber: vLog.BlockNumber,
		EventSig:    "Staked",
		User:        user.Hex(),
		Amount:      amount.String(),
	}
	if err := l.producer.Publish(ctx, msg); err != nil {
		slog.Error("failed to publish Staked", "error", err)
	}
}

func (l *Listener) publishWithdrawn(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 3 {
		return
	}
	user := common.BytesToAddress(vLog.Topics[1].Bytes())
	amount := new(big.Int).SetBytes(vLog.Topics[2].Bytes())

	slog.Info("Withdrawn event", "user", user.Hex(), "amount", amount.String())

	msg := &mq.EventMessage{
		TxHash:      vLog.TxHash.Hex(),
		LogIndex:    vLog.Index,
		BlockNumber: vLog.BlockNumber,
		EventSig:    "Withdrawn",
		User:        user.Hex(),
		Amount:      amount.String(),
	}
	if err := l.producer.Publish(ctx, msg); err != nil {
		slog.Error("failed to publish Withdrawn", "error", err)
	}
}

func (l *Listener) publishClaimed(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 3 {
		return
	}
	user := common.BytesToAddress(vLog.Topics[1].Bytes())
	reward := new(big.Int).SetBytes(vLog.Topics[2].Bytes())

	slog.Info("Claimed event", "user", user.Hex(), "reward", reward.String())

	msg := &mq.EventMessage{
		TxHash:      vLog.TxHash.Hex(),
		LogIndex:    vLog.Index,
		BlockNumber: vLog.BlockNumber,
		EventSig:    "Claimed",
		User:        user.Hex(),
		Amount:      reward.String(),
	}
	if err := l.producer.Publish(ctx, msg); err != nil {
		slog.Error("failed to publish Claimed", "error", err)
	}
}
