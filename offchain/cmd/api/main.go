package main

import (
	"fmt"
	"log/slog"
	"math/big"
	"os"

	"staking-offchain/internal/api/handler"
	"staking-offchain/internal/api/service"
	"staking-offchain/internal/config"
	"staking-offchain/internal/contract"
	"staking-offchain/internal/indexer/repository"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 设置 JSON 日志格式
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 测试：看这行是不是 JSON
	slog.Info("TEST: this should be JSON")

	// 1. 加载配置
	cfg := config.Load()

	// 打印配置确认（调试用）
	slog.Info("Config loaded",
		"rpc_url", cfg.RPCURL,
		"contract_addr", cfg.ContractAddr,
		"private_key_prefix", cfg.PrivateKey[:10],
		"chain_id", cfg.ChainID,
	)

	// 2. 连接 DB
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect DB", "error", err)
		os.Exit(1)
	}

	// 3. 连接链节点
	ethClient, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		slog.Error("Failed to connect Ethereum", "error", err)
		os.Exit(1)
	}

	// 4. 加载私钥
	privateKey, err := crypto.HexToECDSA(cfg.PrivateKey)
	if err != nil {
		slog.Error("Failed to parse private key", "error", err)
		os.Exit(1)
	}

	// 5. 创建 TransactOpts（用于发交易）
	chainID := big.NewInt(cfg.ChainID)
	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		slog.Error("Failed to create transactor", "error", err)
		os.Exit(1)
	}
	transactOpts.GasLimit = uint64(300000)
	transactOpts.GasPrice = big.NewInt(1000000000) // 1 Gwei

	// 打印 transactOpts 确认
	slog.Info("TransactOpts created",
		"from", transactOpts.From.Hex(),
		"gas_limit", transactOpts.GasLimit,
		"gas_price", transactOpts.GasPrice,
	)

	// 6. 实例化合约
	contractAddr := common.HexToAddress(cfg.ContractAddr)
	contractInstance, err := contract.NewContract(contractAddr, ethClient)
	if err != nil {
		slog.Error("Failed to instantiate contract", "error", err)
		os.Exit(1)
	}

	// 7. 组装依赖
	repo := repository.NewRepositoryFromDB(db)
	stakeService := service.NewStakeService(repo, contractInstance, transactOpts)
	stakeHandler := handler.NewStakeHandler(stakeService)
	healthHandler := handler.NewHealthHandler() // 新增

	// 8. 配置路由
	r := gin.Default()

	// 健康检查（独立）
	r.GET("/health", healthHandler.Health)

	api := r.Group("/api/v1")
	{
		// 查询（已有）
		api.GET("/stake/:address", stakeHandler.GetStakeInfo)
		api.GET("/stake/:address/claims", stakeHandler.GetClaims)

		// 操作（新增）
		api.POST("/stake", stakeHandler.Stake)
		api.POST("/withdraw", stakeHandler.Withdraw)
		api.POST("/claim", stakeHandler.Claim)
	}

	// 9. 启动服务
	slog.Info("API server starting", "port", cfg.APIPort)
	r.Run(":" + cfg.APIPort)
}
