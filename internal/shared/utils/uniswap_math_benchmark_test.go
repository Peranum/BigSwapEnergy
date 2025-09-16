package utils

import (
	"math/big"
	"testing"
)

func TestCalculateUniswapV2SwapAmount(t *testing.T) {
	reserveIn := big.NewInt(1_000_000)
	reserveOut := big.NewInt(1_000_000)
	amountIn := big.NewInt(1_000)

	pool := NewBigIntPool()
	defer func() {
		for i := 0; i < 10; i++ {
			if obj := pool.Get(); obj != nil {
				pool.Put(obj)
			}
		}
	}()

	amountOut := CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, pool)

	amountInWithFee := new(big.Int).Mul(amountIn, big.NewInt(997))
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(big.Int).Mul(reserveIn, big.NewInt(1000))
	denominator.Add(denominator, amountInWithFee)
	expected := new(big.Int).Div(numerator, denominator)

	if amountOut.Cmp(expected) != 0 {
		t.Fatalf("unexpected result: got %s want %s", amountOut.String(), expected.String())
	}

	if amountOut.Sign() <= 0 {
		t.Fatalf("amountOut should be positive, got %s", amountOut.String())
	}

	if amountOut.Cmp(amountIn) >= 0 {
		t.Fatalf("amountOut should be less than amountIn due to fee, got %s >= %s", amountOut.String(), amountIn.String())
	}
}

func BenchmarkCalculateUniswapV2SwapAmount_NoAlloc(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)
	amountIn := new(big.Int).SetUint64(1_000_000)

	pool := NewBigIntPool()
	defer func() {
		for i := 0; i < 20; i++ {
			if obj := pool.Get(); obj != nil {
				pool.Put(obj)
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, pool)
	}
}

func BenchmarkCalculateUniswapV2SwapAmount_WithAlloc(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)
	amountIn := new(big.Int).SetUint64(1_000_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pool := NewBigIntPool()
		_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, pool)
	}
}

func BenchmarkCalculateSwapAmount_Direct(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)
	amountIn := new(big.Int).SetUint64(1_000_000)

	pool := NewBigIntPool()
	defer func() {
		for i := 0; i < 20; i++ {
			if obj := pool.Get(); obj != nil {
				pool.Put(obj)
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CalculateSwapAmount(amountIn, reserveIn, reserveOut, 3, pool)
	}
}

func BenchmarkMultipleSwaps_WithPool(b *testing.B) {
	pool := NewBigIntPool()
	defer func() {
		for i := 0; i < 100; i++ {
			if obj := pool.Get(); obj != nil {
				pool.Put(obj)
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reserveIn := new(big.Int).SetUint64(13_451_234_567_890 + uint64(i%1000))
		reserveOut := new(big.Int).SetUint64(98_765_432_109_876 + uint64(i%1000))
		amountIn := new(big.Int).SetUint64(1_000_000 + uint64(i%100))

		_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, pool)
	}
}

func BenchmarkMultipleSwaps_WithoutPool(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reserveIn := new(big.Int).SetUint64(13_451_234_567_890 + uint64(i%1000))
		reserveOut := new(big.Int).SetUint64(98_765_432_109_876 + uint64(i%1000))
		amountIn := new(big.Int).SetUint64(1_000_000 + uint64(i%100))

		pool := NewBigIntPool()
		_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, pool)
	}
}

func BenchmarkBatchProcessing_WithPool(b *testing.B) {
	pool := NewBigIntPool()
	defer func() {
		for i := 0; i < 200; i++ {
			if obj := pool.Get(); obj != nil {
				pool.Put(obj)
			}
		}
	}()

	batchSize := 10
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for j := 0; j < batchSize; j++ {
			reserveIn := new(big.Int).SetUint64(13_451_234_567_890 + uint64(j))
			reserveOut := new(big.Int).SetUint64(98_765_432_109_876 + uint64(j))
			amountIn := new(big.Int).SetUint64(1_000_000 + uint64(j*100))

			_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, pool)
		}
	}
}

func BenchmarkBatchProcessing_WithoutPool(b *testing.B) {
	batchSize := 10
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for j := 0; j < batchSize; j++ {
			reserveIn := new(big.Int).SetUint64(13_451_234_567_890 + uint64(j))
			reserveOut := new(big.Int).SetUint64(98_765_432_109_876 + uint64(j))
			amountIn := new(big.Int).SetUint64(1_000_000 + uint64(j*100))

			pool := NewBigIntPool()
			_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, pool)
		}
	}
}
