package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"
)

var (
	ErrConnectionFailed  = fmt.Errorf("Unable to connect to blockchain network")
	ErrInvalidAddress    = fmt.Errorf("Invalid Ethereum address")
	ErrRPCTimeout        = fmt.Errorf("Blockchain network timeout")
	ErrStorageReadFailed = fmt.Errorf("Unable to read contract data")
)

type EthereumClient interface {
	// GetLatestBlockNumber returns the number of the latest block
	GetLatestBlockNumber(ctx context.Context) (uint64, error)

	// ReadContractStorage reads data from contract storage at specific slot
	ReadContractStorage(ctx context.Context, contractAddress common.Address, storageKey common.Hash, blockNumber *big.Int) ([]byte, error)

	// Close gracefully closes the connection
	Close() error

	// CheckConnectionHealth verifies health status of the connection
	CheckConnectionHealth(ctx context.Context) bool
}

// OptimizedEthereumClient implements EthereumClient interface with optimized HTTP connection pooling
type OptimizedEthereumClient struct {
	client *ethclient.Client
	logger *zap.Logger
	rpcURL string
}

// NewEthereumClient creates a new Ethereum client with optimized HTTP connection pooling
func NewEthereumClient(rpcURL string, logger *zap.Logger) (EthereumClient, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
		MaxConnsPerHost:     0,
		DisableKeepAlives:   false,
		DisableCompression:  false,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	rpcClient, err := rpc.DialOptions(context.Background(), rpcURL, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	client := ethclient.NewClient(rpcClient)

	logger.Info("Created optimized Ethereum client",
		zap.String("url", rpcURL),
		zap.Int("max_idle_conns", transport.MaxIdleConns),
		zap.Int("max_idle_conns_per_host", transport.MaxIdleConnsPerHost))

	return &OptimizedEthereumClient{
		client: client,
		logger: logger,
		rpcURL: rpcURL,
	}, nil
}

// GetLatestBlockNumber returns the number of the latest block using optimized HTTP connection pooling
func (c *OptimizedEthereumClient) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	blockNumber, err := c.client.BlockNumber(ctx)
	if err != nil {
		if isTimeoutError(err) {
			return 0, fmt.Errorf("%w: %v", ErrRPCTimeout, err)
		}
		return 0, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	return blockNumber, nil
}

// ReadContractStorage reads data from contract storage at specific slot using optimized HTTP connection pooling
func (c *OptimizedEthereumClient) ReadContractStorage(ctx context.Context, contractAddress common.Address, storageKey common.Hash, blockNumber *big.Int) ([]byte, error) {
	data, err := c.client.StorageAt(ctx, contractAddress, storageKey, blockNumber)
	if err != nil {
		if isTimeoutError(err) {
			return nil, fmt.Errorf("%w: %v", ErrRPCTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrStorageReadFailed, err)
	}
	return data, nil
}

// Close gracefully closes the connection
func (c *OptimizedEthereumClient) Close() error {
	c.client.Close()
	c.logger.Info("Closed optimized Ethereum client")
	return nil
}

// CheckConnectionHealth verifies health status of the connection
func (c *OptimizedEthereumClient) CheckConnectionHealth(ctx context.Context) bool {
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.BlockNumber(healthCtx)
	return err == nil
}

// isTimeoutError checks if the error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return err == context.DeadlineExceeded || err == context.Canceled
}
