
.PHONY: benchmark uniswap-math-benchmark allocations-benchmark memory-pressure-benchmark

uniswap-math-benchmark:
	@echo "Running Uniswap V2 math benchmarks..."
	@go test -bench=BenchmarkUniswapV2 -benchmem -benchtime=10s ./internal/shared/utils/

allocations-benchmark:
	@echo "Running memory allocation benchmarks..."
	@go test -bench=BenchmarkAllocations -benchmem -benchtime=5s ./internal/shared/utils/

memory-pressure-benchmark:
	@echo "Testing memory pressure scenarios..."
	@go test -bench=BenchmarkMemoryPressure -benchmem -benchtime=5s ./internal/shared/utils/

allocations-test:
	@echo "Testing CalculateUniswapV2SwapAmount allocations..."
	@go test -bench=BenchmarkCalculateUniswapV2SwapAmountAllocations -benchmem -benchtime=5s ./internal/shared/utils/
