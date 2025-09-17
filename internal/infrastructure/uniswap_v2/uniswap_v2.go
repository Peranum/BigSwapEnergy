package uniswap_v2

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"bigswapenergy/internal/infrastructure/ethereum"
	"bigswapenergy/internal/shared/utils"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

const (
	UniswapV2Token0StorageSlot   = 6
	UniswapV2Token1StorageSlot   = 7
	UniswapV2ReservesStorageSlot = 8
)

var (
	ZeroAddress = common.Address{}
)

var (
	ErrPoolNotFound          = fmt.Errorf("Pool not found")
	ErrInsufficientLiquidity = fmt.Errorf("Insufficient liquidity in pool")
	ErrTokenPairMismatch     = fmt.Errorf("Token pair does not match pool")
	ErrInvalidPoolAddress    = fmt.Errorf("Invalid pool address")
)

// UniswapV2Client defines the interface for Uniswap V2 operations
type UniswapV2Client interface {
	// ReadStorageSlot reads a storage slot from the contract
	ReadStorageSlot(ctx context.Context, pool common.Address, blockNum *big.Int, slot uint64) ([]byte, error)

	// GetLatestBlockNumber returns the number of the latest block
	GetLatestBlockNumber(ctx context.Context) (uint64, error)

	// LoadTokens reads token0 and token1 from Uniswap V2 pair storage
	LoadTokens(ctx context.Context, pool common.Address, blockNum *big.Int) (common.Address, common.Address, error)

	// LoadReserves reads reserves from Uniswap V2 pair storage
	LoadReserves(ctx context.Context, pool common.Address, blockNum *big.Int) (*big.Int, *big.Int, error)

	// DetermineReserveOrder determines which reserve corresponds to src and dst tokens
	DetermineReserveOrder(src, dst, token0, token1 common.Address, reserve0, reserve1 *big.Int) (*big.Int, *big.Int, error)
}

// UniswapV2ClientImpl implements Uniswap V2 operations
type UniswapV2ClientImpl struct {
	client ethereum.EthereumClient
	logger *zap.Logger
}

// NewUniswapV2Client creates a new Uniswap V2 client
func NewUniswapV2Client(client ethereum.EthereumClient, logger *zap.Logger) UniswapV2Client {
	return &UniswapV2ClientImpl{
		client: client,
		logger: logger,
	}
}

// ReadStorageSlot reads a storage slot from the contract
func (c *UniswapV2ClientImpl) ReadStorageSlot(ctx context.Context, pool common.Address, blockNum *big.Int, slot uint64) ([]byte, error) {
	var key common.Hash
	key[31] = byte(slot)
	return c.client.ReadContractStorage(ctx, pool, key, blockNum)
}

// GetLatestBlockNumber returns the number of the latest block
func (c *UniswapV2ClientImpl) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	return c.client.GetLatestBlockNumber(ctx)
}

// LoadTokens reads token0 and token1 from Uniswap V2 pair storage
func (c *UniswapV2ClientImpl) LoadTokens(ctx context.Context, pool common.Address, blockNum *big.Int) (common.Address, common.Address, error) {
	token0Data, err := c.ReadStorageSlot(ctx, pool, blockNum, UniswapV2Token0StorageSlot)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("failed to read token0: %w", err)
	}
	token0 := common.BytesToAddress(token0Data)

	token1Data, err := c.ReadStorageSlot(ctx, pool, blockNum, UniswapV2Token1StorageSlot)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("failed to read token1: %w", err)
	}
	token1 := common.BytesToAddress(token1Data)

	if bytes.Equal(token0[:], ZeroAddress[:]) || bytes.Equal(token1[:], ZeroAddress[:]) {
		return common.Address{}, common.Address{}, fmt.Errorf("%w for pool %s", ErrPoolNotFound, pool.Hex())
	}

	return token0, token1, nil
}

// LoadReserves reads reserves from Uniswap V2 pair storage
func (c *UniswapV2ClientImpl) LoadReserves(ctx context.Context, pool common.Address, blockNum *big.Int) (*big.Int, *big.Int, error) {
	reserveData, err := c.ReadStorageSlot(ctx, pool, blockNum, UniswapV2ReservesStorageSlot)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read reserves: %w", err)
	}

	reserve0, reserve1 := utils.ParseReserves(reserveData)

	if reserve0.Sign() == 0 || reserve1.Sign() == 0 {
		return nil, nil, fmt.Errorf("%w for pool %s", ErrInsufficientLiquidity, pool.Hex())
	}

	return reserve0, reserve1, nil
}

// DetermineReserveOrder determines which reserve corresponds to src and dst tokens
func (c *UniswapV2ClientImpl) DetermineReserveOrder(src, dst, token0, token1 common.Address, reserve0, reserve1 *big.Int) (*big.Int, *big.Int, error) {
	switch {
	case bytes.Equal(src[:], token0[:]) && bytes.Equal(dst[:], token1[:]):
		return reserve0, reserve1, nil
	case bytes.Equal(src[:], token1[:]) && bytes.Equal(dst[:], token0[:]):
		return reserve1, reserve0, nil
	default:
		return nil, nil, fmt.Errorf("%w: src=%s dst=%s token0=%s token1=%s",
			ErrTokenPairMismatch, src.Hex(), dst.Hex(), token0.Hex(), token1.Hex())
	}
}
