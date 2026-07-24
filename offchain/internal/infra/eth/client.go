// internal/infra/eth/client.go
package eth

import (
	"log"
	"math/big"

	"staking-offchain/internal/contract"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	EthClient *ethclient.Client
	Filterer  *contract.ContractFilterer // 使用 ContractFilterer
	Auth      *bind.TransactOpts         // 修正：TransactOpts（不是 Transact0pts）
}

func NewClient(rpcURL string, contractAddr string, privateKey string, chainID int64) *Client { // 修正：*Client（不是 *client）
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// 1. Filterer（给 Indexer 用，监听事件）
	filterer, err := contract.NewContractFilterer(
		common.HexToAddress(contractAddr),
		client,
	)
	if err != nil {
		log.Fatalf("Failed to create filterer: %v", err)
	}

	// 2. Auth（给 API 用，发交易）
	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(chainID))
	if err != nil {
		log.Fatalf("Failed to create transactor: %v", err)
	}

	return &Client{
		EthClient: client,
		Filterer:  filterer,
		Auth:      auth,
	}
}
