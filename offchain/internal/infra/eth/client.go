package eth

import (
	"log/slog"
	"math/big"
	"os"

	"staking-offchain/internal/contract"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	EthClient *ethclient.Client
	Filterer  *contract.ContractFilterer
	Auth      *bind.TransactOpts
}

func NewClient(rpcURL string, contractAddr string, privateKey string, chainID int64) *Client {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		slog.Error("failed to connect ethereum", "error", err)
		os.Exit(1)
	}

	filterer, err := contract.NewContractFilterer(
		common.HexToAddress(contractAddr),
		client,
	)
	if err != nil {
		slog.Error("failed to create filterer", "error", err)
		os.Exit(1)
	}

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		slog.Error("invalid private key", "error", err)
		os.Exit(1)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(chainID))
	if err != nil {
		slog.Error("failed to create transactor", "error", err)
		os.Exit(1)
	}

	return &Client{
		EthClient: client,
		Filterer:  filterer,
		Auth:      auth,
	}
}
