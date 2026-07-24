package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"staking-offchain/internal/config"
	"staking-offchain/internal/indexer"
	"staking-offchain/internal/indexer/repository"
	"staking-offchain/internal/infra/eth"
)

func main() {
	// 设置 JSON 日志格式
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	_ = godotenv.Load()

	cfg := config.Load()

	// 临时调试：看看读出来的地址到底是什么
	slog.Info("DEBUG: loaded config", "contract_addr", cfg.ContractAddr)

	slog.Info("Starting Staking Indexer",
		"rpc_url", cfg.RPCURL,
		"contract_addr", cfg.ContractAddr,
		"chain_id", cfg.ChainID,
	)

	ethClient := eth.NewClient(cfg.RPCURL, cfg.ContractAddr, cfg.PrivateKey, cfg.ChainID)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	repo, err := repository.NewRepository(dsn)
	if err != nil {
		slog.Error("Failed to create repository", "error", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, err := repo.GetDB().DB()
		if err != nil {
			slog.Warn("Failed to close DB", "error", err)
		} else {
			sqlDB.Close()
		}
	}()

	listener := indexer.NewListener(ethClient, repo, cfg.ContractAddr, cfg.ChainID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down...")
		cancel()
	}()

	listener.Start(ctx)
}
