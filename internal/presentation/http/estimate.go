package http

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"bigswapenergy/internal/shared/config"
	apperrors "bigswapenergy/internal/shared/errors"
	estimate "bigswapenergy/internal/usecases"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type EstimateHandler struct {
	estimateService estimate.EstimateService
	logger          *zap.Logger
	config          *config.Config
}

// GetRateLimitConfig implements RateLimitable interface
func (h *EstimateHandler) GetRateLimitConfig() HTTPRateLimitConfig {
	return HTTPRateLimitConfig{
		RequestsPerMinute: h.config.RateLimit.RequestsPerMinute,
	}
}

func NewEstimateHandler(estimateService estimate.EstimateService, logger *zap.Logger, config *config.Config) *EstimateHandler {
	return &EstimateHandler{
		estimateService: estimateService,
		logger:          logger,
		config:          config,
	}
}

// EstimateSwapAmount handles the /estimate endpoint
func (h *EstimateHandler) EstimateSwapAmount(ctx *fasthttp.RequestCtx) {
	startTime := time.Now()

	poolAddress, srcToken, dstToken, srcAmountBig, err := h.parseEstimateParams(ctx)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	dstAmount, err := h.estimateService.EstimateSwapAmount(ctx, poolAddress, srcToken, dstToken, srcAmountBig)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	totalDuration := time.Since(startTime)

	h.logger.Info("Estimate completed", zap.Duration("duration", totalDuration))

	ctx.SetContentType("text/plain")
	dstAmountStr := dstAmount.String()
	ctx.SetBodyString(dstAmountStr)
}

func (h *EstimateHandler) parseEstimateParams(ctx *fasthttp.RequestCtx) (string, string, string, *big.Int, error) {
	poolBytes := ctx.QueryArgs().Peek("pool")
	if len(poolBytes) == 0 {
		return "", "", "", nil, fmt.Errorf("%w: pool parameter is required", apperrors.ErrValidation)
	}
	poolValue := string(poolBytes)

	srcBytes := ctx.QueryArgs().Peek("src")
	if len(srcBytes) == 0 {
		return "", "", "", nil, fmt.Errorf("%w: source token parameter is required", apperrors.ErrValidation)
	}
	srcValue := string(srcBytes)

	dstBytes := ctx.QueryArgs().Peek("dst")
	if len(dstBytes) == 0 {
		return "", "", "", nil, fmt.Errorf("%w: destination token parameter is required", apperrors.ErrValidation)
	}
	dstValue := string(dstBytes)

	srcAmountBytes := ctx.QueryArgs().Peek("src_amount")
	if len(srcAmountBytes) == 0 {
		return "", "", "", nil, fmt.Errorf("%w: source amount parameter is required", apperrors.ErrValidation)
	}
	srcAmountValue := string(srcAmountBytes)

	srcAmount, err := strconv.ParseInt(srcAmountValue, 10, 64)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("%w: source amount must be a valid number", apperrors.ErrValidation)
	}

	if srcAmount <= 0 {
		return "", "", "", nil, fmt.Errorf("%w: source amount must be positive", apperrors.ErrValidation)
	}

	srcAmountBig := big.NewInt(srcAmount)

	return poolValue, srcValue, dstValue, srcAmountBig, nil
}
