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
	"staking-offchain/internal/dao"
	"staking-offchain/internal/mq"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	_ = godotenv.Load()

	cfg := config.Load()

	slog.Info("starting consumer", "queue", cfg.MQQueue)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := dao.InitDB(dsn)
	if err != nil {
		slog.Error("failed to init db", "error", err)
		os.Exit(1)
	}

	consumer, err := mq.NewConsumer(cfg.MQURL, cfg.MQQueue, db)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("shutting down...")
		cancel()
	}()

	if err := consumer.Start(ctx); err != nil {
		slog.Error("consumer error", "error", err)
	}
}
