package estimate

import (
	"context"
	"fmt"
	"math/big"

	"bigswapenergy/internal/infrastructure/uniswap_v2"
	apperrors "bigswapenergy/internal/shared/errors"
	"bigswapenergy/internal/shared/utils"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

// EstimateService defines the interface for swap estimation operations
type EstimateService interface {
	// EstimateSwapAmount calculates the estimated destination amount for a Uniswap V2 swap
	// based on the latest blockchain state
	EstimateSwapAmount(ctx context.Context, poolAddress, srcToken, dstToken string, srcAmount *big.Int) (*big.Int, error)
}

// EstimateServiceImpl implements swap estimation operations
type EstimateServiceImpl struct {
	uniswapV2Client uniswap_v2.UniswapV2Client
	logger          *zap.Logger
}

// NewEstimateService creates a new estimate service
func NewEstimateService(
	uniswapV2Client uniswap_v2.UniswapV2Client,
	logger *zap.Logger,
) EstimateService {
	return &EstimateServiceImpl{
		uniswapV2Client: uniswapV2Client,
		logger:          logger,
	}
}

// EstimateSwapAmount calculates the estimated destination amount for a Uniswap V2 swap
// based on the latest blockchain state
func (s *EstimateServiceImpl) EstimateSwapAmount(ctx context.Context, poolAddress, srcToken, dstToken string, srcAmount *big.Int) (*big.Int, error) {
	if poolAddress == "" {
		return nil, fmt.Errorf("%w: pool address is required", apperrors.ErrValidation)
	}
	if srcToken == "" {
		return nil, fmt.Errorf("%w: source token address is required", apperrors.ErrValidation)
	}
	if dstToken == "" {
		return nil, fmt.Errorf("%w: destination token address is required", apperrors.ErrValidation)
	}
	if srcAmount == nil || srcAmount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: source amount must be positive", apperrors.ErrValidation)
	}

	srcAmountStr := srcAmount.String()
	s.logger.Info("Processing swap estimation request",
		zap.String("pool", poolAddress),
		zap.String("src_token", srcToken),
		zap.String("dst_token", dstToken),
		zap.String("src_amount", srcAmountStr),
	)

	if !common.IsHexAddress(poolAddress) {
		return nil, fmt.Errorf("%w: invalid pool address format: %s", apperrors.ErrValidation, poolAddress)
	}
	if !common.IsHexAddress(srcToken) {
		return nil, fmt.Errorf("%w: invalid source token address format: %s", apperrors.ErrValidation, srcToken)
	}
	if !common.IsHexAddress(dstToken) {
		return nil, fmt.Errorf("%w: invalid destination token address format: %s", apperrors.ErrValidation, dstToken)
	}

	pool := common.HexToAddress(poolAddress)
	src := common.HexToAddress(srcToken)
	dst := common.HexToAddress(dstToken)

	if src == dst {
		return nil, fmt.Errorf("%w: source and destination tokens cannot be the same", apperrors.ErrBusinessRule)
	}

	blockNumber, err := s.uniswapV2Client.GetLatestBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to connect to blockchain network: %v", apperrors.ErrExternalService, err)
	}
	blockNum := utils.GlobalBigIntPool.Get()
	blockNum.SetUint64(blockNumber)
	defer utils.GlobalBigIntPool.Put(blockNum)

	token0, token1, err := s.uniswapV2Client.LoadTokens(ctx, pool, blockNum)
	if err != nil {
		return nil, fmt.Errorf("%w: pool not found or invalid: %v", apperrors.ErrNotFound, err)
	}

	reserve0, reserve1, err := s.uniswapV2Client.LoadReserves(ctx, pool, blockNum)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to read pool reserves: %v", apperrors.ErrExternalService, err)
	}

	reserveIn, reserveOut, err := s.uniswapV2Client.DetermineReserveOrder(src, dst, token0, token1, reserve0, reserve1)
	if err != nil {
		return nil, err
	}

	if reserveIn.Sign() == 0 || reserveOut.Sign() == 0 {
		return nil, fmt.Errorf("%w: pool has empty reserves", apperrors.ErrBusinessRule)
	}

	amountOut := utils.CalculateUniswapV2SwapAmount(srcAmount, reserveIn, reserveOut, utils.GlobalBigIntPool)

	return amountOut, nil
}
