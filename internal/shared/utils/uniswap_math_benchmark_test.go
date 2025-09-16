package utils

import (
	"math/big"
	"sync"
	"testing"
)

func TestCalculateUniswapV2SwapAmount(t *testing.T) {
	reserveIn := big.NewInt(1_000_000)
	reserveOut := big.NewInt(1_000_000)
	amountIn := big.NewInt(1_000)

	amountOut := CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, GlobalBigIntPool)

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

// BenchmarkSingleRequest tests a single request using global pool (production scenario)
func BenchmarkSingleRequest(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)
	amountIn := new(big.Int).SetUint64(1_000_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, GlobalBigIntPool)
	}
}

// BenchmarkConcurrentRequests tests 10 concurrent requests (typical production load)
func BenchmarkConcurrentRequests(b *testing.B) {
	numGoroutines := 10
	operationsPerGoroutine := b.N / numGoroutines

	b.ReportAllocs()
	b.ResetTimer()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < operationsPerGoroutine; i++ {
				reserveIn := new(big.Int).SetUint64(13_451_234_567_890 + uint64(i%1000))
				reserveOut := new(big.Int).SetUint64(98_765_432_109_876 + uint64(i%1000))
				amountIn := new(big.Int).SetUint64(1_000_000 + uint64(i%100))

				_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, GlobalBigIntPool)
			}
		}(g)
	}

	wg.Wait()
}

// BenchmarkHighLoad tests high load scenario with many concurrent requests
func BenchmarkHighLoad(b *testing.B) {
	numGoroutines := 50
	operationsPerGoroutine := b.N / numGoroutines

	b.ReportAllocs()
	b.ResetTimer()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < operationsPerGoroutine; i++ {
				reserveIn := new(big.Int).SetUint64(13_451_234_567_890 + uint64(i%1000))
				reserveOut := new(big.Int).SetUint64(98_765_432_109_876 + uint64(i%1000))
				amountIn := new(big.Int).SetUint64(1_000_000 + uint64(i%100))

				_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, GlobalBigIntPool)
			}
		}(g)
	}

	wg.Wait()
}

// TestGlobalPoolIntegration tests that the global pool works correctly in real scenarios
func TestGlobalPoolIntegration(t *testing.T) {
	reserveIn := big.NewInt(1_000_000)
	reserveOut := big.NewInt(1_000_000)
	amountIn := big.NewInt(1_000)

	var results []*big.Int
	for i := 0; i < 10; i++ {
		amountOut := CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, GlobalBigIntPool)
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

// BenchmarkAllocationsWithPool tests memory allocations when using the pool
func BenchmarkAllocationsWithPool(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)
	amountIn := new(big.Int).SetUint64(1_000_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, GlobalBigIntPool)
	}
}

// BenchmarkAllocationsWithoutPool tests memory allocations without using the pool
func BenchmarkAllocationsWithoutPool(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)
	amountIn := new(big.Int).SetUint64(1_000_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		numerator := new(big.Int).Mul(amountIn, big.NewInt(997))
		tmpCalc := new(big.Int).Mul(reserveIn, big.NewInt(1000))
		denominator := new(big.Int).Add(tmpCalc, numerator)
		tmpCalc.Mul(numerator, reserveOut)
		result := new(big.Int).Div(tmpCalc, denominator)
		_ = result
	}
}

// BenchmarkAllocationsComparison compares allocations with and without pool
func BenchmarkAllocationsComparison(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)
	amountIn := new(big.Int).SetUint64(1_000_000)

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = CalculateUniswapV2SwapAmount(amountIn, reserveIn, reserveOut, GlobalBigIntPool)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			numerator := new(big.Int).Mul(amountIn, big.NewInt(997))
			tmpCalc := new(big.Int).Mul(reserveIn, big.NewInt(1000))
			denominator := new(big.Int).Add(tmpCalc, numerator)
			tmpCalc.Mul(numerator, reserveOut)
			result := new(big.Int).Div(tmpCalc, denominator)
			_ = result
		}
	})
}

// BenchmarkMemoryPressure tests behavior under memory pressure
func BenchmarkMemoryPressure(b *testing.B) {
	reserveIn := new(big.Int).SetUint64(13_451_234_567_890)
	reserveOut := new(big.Int).SetUint64(98_765_432_109_876)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		amountInVaried := new(big.Int).SetUint64(1_000_000 + uint64(i%1000))
		_ = CalculateUniswapV2SwapAmount(amountInVaried, reserveIn, reserveOut, GlobalBigIntPool)
	}
}
