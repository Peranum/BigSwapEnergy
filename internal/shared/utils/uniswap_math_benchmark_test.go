package utils

import (
	"math/big"
	"testing"
)

func TestCalculateUniswapV2SwapAmount(t *testing.T) {
	reserveIn := big.NewInt(1_000_000)
	reserveOut := big.NewInt(1_000_000)
	amountIn := big.NewInt(1_000)

	amountOut := GlobalBigIntPool.Get()
	CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, amountOut, GlobalBigIntPool)

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

// TestGlobalPoolIntegration tests that the global pool works correctly in real scenarios
func TestGlobalPoolIntegration(t *testing.T) {
	reserveIn := big.NewInt(1_000_000)
	reserveOut := big.NewInt(1_000_000)
	amountIn := big.NewInt(1_000)

	var results []*big.Int
	for i := 0; i < 10; i++ {
		amountOut := GlobalBigIntPool.Get()
		CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, amountOut, GlobalBigIntPool)
		results = append(results, amountOut)
	}

	for i := 1; i < len(results); i++ {
		if results[i].Cmp(results[0]) != 0 {
			t.Fatalf("Global pool produced inconsistent results: %s vs %s",
				results[i].String(), results[0].String())
		}
	}

	expected := big.NewInt(996)
	if results[0].Cmp(expected) != 0 {
		t.Logf("Expected approximately %s, got %s", expected.String(), results[0].String())
	}
}

// BenchmarkCalculateUniswapV2SwapAmountAllocations tests memory allocations in CalculateUniswapV2SwapAmount
func BenchmarkCalculateUniswapV2SwapAmountAllocations(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)
	amountIn := new(big.Int).SetUint64(1_000_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := GlobalBigIntPool.Get()
		CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, result, GlobalBigIntPool)
		GlobalBigIntPool.Put(result)
	}
}

// BenchmarkMemoryPressure tests behavior under memory pressure
func BenchmarkMemoryPressure(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		amountInVaried := new(big.Int).SetUint64(1_000_000 + uint64(i%1000))
		result := GlobalBigIntPool.Get()
		CalculateUniswapV2SwapAmount(amountInVaried, reserveIn, reserveOut, result, GlobalBigIntPool)
		GlobalBigIntPool.Put(result)
	}
}
