
.PHONY: benchmark uniswap-math-benchmark


uniswap-math-benchmark:
	@echo "Running Uniswap V2 math benchmarks..."
	@go test -bench=BenchmarkUniswapV2 -benchmem -benchtime=10s ./internal/shared/utils/
