package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	RPCURL       string
	ContractAddr string
	PrivateKey   string
	ChainID      int64

	APIPort string
	Env     string

	MQURL   string // ← 新增 for RabbitMQ connection URL
	MQQueue string // ← 新增 for RabbitMQ queue name
}

func Load() *Config {
	// 尝试多个可能的 .env 路径
	envPaths := []string{
		"offchain/.env",                         // 从项目根目录运行
		".env",                                  // 当前目录
		"../offchain/.env",                      // 从 offchain 内部运行
		filepath.Join("..", "offchain", ".env"), // 备用
	}

	loaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("✅ Loaded .env from: %s", path)
			loaded = true
			break
		}
	}

	if !loaded {
		log.Println("⚠️ No .env file found, using environment variables")
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "staking"),

		RPCURL:       getEnv("RPC_URL", "http://localhost:8545"),
		ContractAddr: getEnv("CONTRACT_ADDRESS", ""),
		PrivateKey:   getEnv("PRIVATE_KEY", ""),
		ChainID:      getEnvAsInt("CHAIN_ID", 31337),

		APIPort: getEnv("API_PORT", "8080"),
		Env:     getEnv("ENVIRONMENT", "development"),

		MQURL:   getEnv("MQ_URL", "amqp://guest:guest@localhost:5672/"), // ← 新增 for RabbitMQ connection URL
		MQQueue: getEnv("MQ_QUEUE", "staking_events"),                   // ← 新增 for RabbitMQ queue name
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		v, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return defaultVal
		}
		return v
	}
	return defaultVal
}
