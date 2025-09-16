package utils

import (
	"math/big"
	"sync"
)

var (
	FeeBasisPoints1000 = big.NewInt(1000)
	FeeBasisPoints997  = big.NewInt(997) // 1000 - 3 (0.3% fee)
	FeeBasisPoints995  = big.NewInt(995) // 1000 - 5 (0.5% fee)
	FeeBasisPoints990  = big.NewInt(990) // 1000 - 10 (1.0% fee)

	Mask112 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 112), big.NewInt(1))

	GlobalBigIntPool = NewBigIntPool()
)

// BigIntPool provides a pool of reusable big.Int objects for memory optimization
type BigIntPool struct {
	pool sync.Pool
}

// NewBigIntPool creates a new BigInt pool
func NewBigIntPool() *BigIntPool {
	return &BigIntPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(big.Int)
			},
		},
	}
}

// Get retrieves a big.Int from the pool
func (p *BigIntPool) Get() *big.Int {
	return p.pool.Get().(*big.Int)
}

// Put returns a big.Int to the pool
func (p *BigIntPool) Put(x *big.Int) {
	if x != nil {
		x.SetInt64(0)
		p.pool.Put(x)
	}
}

// GetPoolStats returns statistics about the pool (for debugging/monitoring)
func (p *BigIntPool) GetPoolStats() map[string]interface{} {
	// Note: sync.Pool doesn't expose internal stats, so we return basic info
	return map[string]interface{}{
		"pool_type":   "sync.Pool",
		"object_type": "big.Int",
		"pre_warmed":  true,
	}
}

// ParseReserves unpacks two uint112 reserves from the 32-byte storage word
// used by Uniswap V2 pairs. The layout is:
//
//	[ 112 bits reserve0 | 112 bits reserve1 | 32 bits timestamp ]
//
// Values are treated as big-endian within the 256-bit word.
func ParseReserves(b []byte) (reserve0, reserve1 *big.Int) {
	v := new(big.Int).SetBytes(b)

	reserve0 = new(big.Int).And(v, Mask112)
	tmp := new(big.Int).Rsh(v, 112)
	reserve1 = new(big.Int).And(tmp, Mask112)
	return
}

// ParseReservesWithPool unpacks two uint112 reserves using a BigInt pool for memory optimization
func ParseReservesWithPool(b []byte, pool *BigIntPool) (reserve0, reserve1 *big.Int) {
	v := pool.Get()
	v.SetBytes(b)

	tmp := pool.Get()
	reserve0 = pool.Get()
	reserve1 = pool.Get()

	reserve0.And(v, Mask112)
	tmp.Rsh(v, 112)
	reserve1.And(tmp, Mask112)

	pool.Put(v)
	pool.Put(tmp)

	result0 := new(big.Int).Set(reserve0)
	result1 := new(big.Int).Set(reserve1)

	pool.Put(reserve0)
	pool.Put(reserve1)

	return result0, result1
}

// CalculateSwapAmount calculates the swap amount using constant product AMM formula with fee
// Formula: amountOut = (amountIn * (1000-fee) * reserveOut) / (reserveIn * 1000 + amountIn * (1000-fee))
// This formula is used by Uniswap V2, SushiSwap, PancakeSwap, and other constant product AMMs
func CalculateSwapAmount(amountIn, reserveIn, reserveOut *big.Int, feeBasisPoints int, pool *BigIntPool) *big.Int {
	numerator := pool.Get()
	denominator := pool.Get()
	tmpCalc := pool.Get()

	switch feeBasisPoints {
	case 3:
		numerator.Mul(amountIn, FeeBasisPoints997)
	case 5:
		numerator.Mul(amountIn, FeeBasisPoints995)
	case 10:
		numerator.Mul(amountIn, FeeBasisPoints990)
	default:
		feeMultiplier := pool.Get()
		feeMultiplier.SetInt64(int64(1000 - feeBasisPoints))
		numerator.Mul(amountIn, feeMultiplier)
		pool.Put(feeMultiplier)
	}

	tmpCalc.Mul(reserveIn, FeeBasisPoints1000)
	denominator.Add(tmpCalc, numerator)

	tmpCalc.Mul(numerator, reserveOut)
	result := tmpCalc.Div(tmpCalc, denominator)

	amountOut := new(big.Int).Set(result)

	pool.Put(numerator)
	pool.Put(denominator)
	pool.Put(tmpCalc)

	return amountOut
}

// CalculateUniswapV2SwapAmount is a convenience wrapper for Uniswap V2 with 0.3% fee
func CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut *big.Int, pool *BigIntPool) *big.Int {
	return CalculateSwapAmount(amountIn, reserveIn, reserveOut, 3, pool) // 0.3% fee
}
