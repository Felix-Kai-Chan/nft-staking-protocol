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
	"staking-offchain/internal/dao"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

func main() {
	// ✅ 全 JSON 结构化日志
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	slog.Info("config loaded",
		"rpc_url", cfg.RPCURL,
		"contract_addr", cfg.ContractAddr,
		"chain_id", cfg.ChainID,
	)

	// 连接 DB（用 dao.InitDB，含 AutoMigrate + 连接池）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := dao.InitDB(dsn)
	if err != nil {
		slog.Error("failed to init db", "error", err)
		os.Exit(1)
	}

	// 连接链节点
	ethClient, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		slog.Error("failed to connect ethereum", "error", err)
		os.Exit(1)
	}

	// 加载私钥
	privateKey, err := crypto.HexToECDSA(cfg.PrivateKey)
	if err != nil {
		slog.Error("failed to parse private key", "error", err)
		os.Exit(1)
	}

	// TransactOpts
	chainID := big.NewInt(cfg.ChainID)
	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		slog.Error("failed to create transactor", "error", err)
		os.Exit(1)
	}
	transactOpts.GasLimit = uint64(300000)
	transactOpts.GasPrice = big.NewInt(1000000000)

	slog.Info("transact opts ready", "from", transactOpts.From.Hex(), "gas_limit", transactOpts.GasLimit)

	// 合约实例
	contractAddr := common.HexToAddress(cfg.ContractAddr)
	contractInstance, err := contract.NewContract(contractAddr, ethClient)
	if err != nil {
		slog.Error("failed to instantiate contract", "error", err)
		os.Exit(1)
	}

	// 组装依赖（DAO 替代 repository）
	stakeDAO := dao.NewStakeDAO(db)
	claimDAO := dao.NewClaimDAO(db)
	stakeService := service.NewStakeService(stakeDAO, claimDAO, contractInstance, transactOpts)
	stakeHandler := handler.NewStakeHandler(stakeService)
	healthHandler := handler.NewHealthHandler()

	// 路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(RequestLogger())

	r.GET("/health", healthHandler.Health)

	api := r.Group("/api/v1")
	{
		api.GET("/stake/:address", stakeHandler.GetStakeInfo)
		api.GET("/stake/:address/claims", stakeHandler.GetClaims)
		api.POST("/stake", stakeHandler.Stake)
		api.POST("/withdraw", stakeHandler.Withdraw)
		api.POST("/claim", stakeHandler.Claim)
	}

	slog.Info("api server starting", "port", cfg.APIPort)
	r.Run(":" + cfg.APIPort)
}

// RequestLogger 用 slog 记录每个请求
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", c.Writer.Size(),
			"client_ip", c.ClientIP(),
		)
	}
}
