package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var (
	ErrConnectionFailed  = fmt.Errorf("Unable to connect to blockchain network")
	ErrInvalidAddress    = fmt.Errorf("Invalid Ethereum address")
	ErrRPCTimeout        = fmt.Errorf("Blockchain network timeout")
	ErrStorageReadFailed = fmt.Errorf("Unable to read contract data")
)

// EthereumClient defines the interface for Ethereum blockchain client with connection pooling
type EthereumClient interface {
	// GetLatestBlockNumber returns the number of the latest block
	GetLatestBlockNumber(ctx context.Context) (uint64, error)

	// ReadContractStorage reads data from contract storage at specific slot
	ReadContractStorage(ctx context.Context, contractAddress common.Address, storageKey common.Hash, blockNumber *big.Int) ([]byte, error)

	// Close gracefully closes all connections in the pool
	Close() error

	// GetConnectionCount returns the number of active connections
	GetConnectionCount() int

	// CheckConnectionsHealth verifies health status of all connections
	CheckConnectionsHealth(ctx context.Context) []bool
}

// ConnectionPool implements EthereumClient interface with connection pooling for high performance
type ConnectionPool struct {
	clients []*ethclient.Client
	logger  *zap.Logger
	rpcURL  string
	mu      sync.RWMutex
	counter uint64

	bigIntPool  sync.Pool
	callMsgPool sync.Pool
}

// NewEthereumClient creates a new Ethereum client with connection pooling
func NewEthereumClient(rpcURL string, poolSize int, logger *zap.Logger) (EthereumClient, error) {
	clients := make([]*ethclient.Client, poolSize)

	for i := 0; i < poolSize; i++ {
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			for j := 0; j < i; j++ {
				clients[j].Close()
			}
			return nil, err
		}
		clients[i] = client
	}

	logger.Info("Created Ethereum connection pool",
		zap.String("url", rpcURL),
		zap.Int("pool_size", poolSize))

	pool := &ConnectionPool{
		clients: clients,
		logger:  logger,
		rpcURL:  rpcURL,
	}

	pool.bigIntPool = sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}

	pool.callMsgPool = sync.Pool{
		New: func() interface{} {
			return &ethereum.CallMsg{}
		},
	}

	go pool.warmupConnections()

	return pool, nil
}

func (p *ConnectionPool) warmupConnections() {
	p.mu.RLock()
	poolSize := len(p.clients)
	p.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(poolSize)

	for i := 0; i < poolSize; i++ {
		go func(index int) {
			defer wg.Done()

			p.mu.RLock()
			client := p.clients[index]
			p.mu.RUnlock()

			_, err := client.BlockNumber(ctx)
			if err != nil {
				p.logger.Warn("Failed to warm up connection",
					zap.Int("connection", index),
					zap.Error(err))
			} else {
				p.logger.Debug("Connection warmed up successfully",
					zap.Int("connection", index))
			}
		}(i)
	}

	wg.Wait()
	p.logger.Info("Connection pool warmup completed")
}

func (p *ConnectionPool) getClient() *ethclient.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()

	index := atomic.AddUint64(&p.counter, 1) % uint64(len(p.clients))
	return p.clients[index]
}

// GetLatestBlockNumber returns the number of the latest block using connection pool
func (p *ConnectionPool) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	client := p.getClient()
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		if isTimeoutError(err) {
			return 0, fmt.Errorf("%w: %v", ErrRPCTimeout, err)
		}
		return 0, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	return blockNumber, nil
}

// ReadContractStorage reads data from contract storage at specific slot using connection pool
func (p *ConnectionPool) ReadContractStorage(ctx context.Context, contractAddress common.Address, storageKey common.Hash, blockNumber *big.Int) ([]byte, error) {
	client := p.getClient()
	data, err := client.StorageAt(ctx, contractAddress, storageKey, blockNumber)
	if err != nil {
		if isTimeoutError(err) {
			return nil, fmt.Errorf("%w: %v", ErrRPCTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrStorageReadFailed, err)
	}
	return data, nil
}

// Close gracefully closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.Close()
	}

	p.logger.Info("Closed Ethereum connection pool")
	return nil
}

// GetConnectionCount returns the number of active connections
func (p *ConnectionPool) GetConnectionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.clients)
}

// CheckConnectionsHealth verifies health status of all connections
func (p *ConnectionPool) CheckConnectionsHealth(ctx context.Context) []bool {
	p.mu.RLock()
	poolSize := len(p.clients)
	p.mu.RUnlock()

	health := make([]bool, poolSize)

	var wg sync.WaitGroup
	wg.Add(poolSize)

	for i := 0; i < poolSize; i++ {
		go func(index int) {
			defer wg.Done()

			p.mu.RLock()
			client := p.clients[index]
			p.mu.RUnlock()

			healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			_, err := client.BlockNumber(healthCtx)
			health[index] = (err == nil)
		}(i)
	}

	wg.Wait()
	return health
}

// isTimeoutError checks if the error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return err == context.DeadlineExceeded || err == context.Canceled
}
