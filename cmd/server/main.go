package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bigswapenergy/internal/infrastructure/ethereum"
	"bigswapenergy/internal/infrastructure/uniswap_v2"
	"bigswapenergy/internal/presentation/http"
	"bigswapenergy/internal/shared/config"
	"bigswapenergy/internal/shared/logger"
	estimate "bigswapenergy/internal/usecases"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func main() {
	log := logger.NewLogger()
	defer log.Sync()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}

	poolSize := 5
	ethClient, err := ethereum.NewEthereumClient(cfg.Blockchain.EthereumRPCURL, poolSize, log)
	if err != nil {
		log.Fatal("Failed to create Ethereum connection pool", zap.Error(err))
	}
	defer ethClient.Close()

	uniswapV2Client := uniswap_v2.NewUniswapV2Client(ethClient, log)

	estimateService := estimate.NewEstimateService(uniswapV2Client, log)

	estimateHandler := http.NewEstimateHandler(estimateService, log, cfg)

	router := setupRouter(estimateHandler, log)

	server := &fasthttp.Server{
		Handler: router,
	}

	serverError := make(chan error, 1)
	go func() {
		log.Info("Starting server", zap.String("address", cfg.Server.Address))
		if err := server.ListenAndServe(cfg.Server.Address); err != nil {
			serverError <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	healthCheckDone := make(chan struct{})
	go func() {
		defer close(healthCheckDone)
		ticker := time.NewTicker(cfg.Server.HealthCheckPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				health := ethClient.CheckConnectionsHealth(ctx)
				cancel()

				healthyCount := 0
				for _, isHealthy := range health {
					if isHealthy {
						healthyCount++
					}
				}

				log.Info("Ethereum connection pool health check",
					zap.Int("healthy", healthyCount),
					zap.Int("total", ethClient.GetConnectionCount()),
					zap.Float64("health_percentage", float64(healthyCount)/float64(ethClient.GetConnectionCount())*100))
			case <-quit:
				log.Info("Health check goroutine stopping")
				return
			}
		}
	}()

	select {
	case <-quit:
		log.Info("Received shutdown signal, starting graceful shutdown")
	case err := <-serverError:
		log.Error("Server error occurred", zap.Error(err))
		log.Info("Starting graceful shutdown due to server error")
	}

	log.Info("Stopping server from accepting new connections")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("Error during server shutdown", zap.Error(err))
	} else {
		log.Info("Server shutdown completed successfully")
	}

	log.Info("Waiting for health check goroutine to finish")
	select {
	case <-healthCheckDone:
		log.Info("Health check goroutine finished")
	case <-time.After(5 * time.Second):
		log.Warn("Health check goroutine did not finish within timeout")
	}

	log.Info("Closing Ethereum connection pool")
	if err := ethClient.Close(); err != nil {
		log.Error("Error closing Ethereum connection pool", zap.Error(err))
	}

}

func setupRouter(estimateHandler *http.EstimateHandler, logger *zap.Logger) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		switch path {
		case "/estimate":
			handler := http.ApplyMiddleware(
				estimateHandler.EstimateSwapAmount,
				logger,
				estimateHandler,
			)
			handler(ctx)
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			ctx.SetBodyString("Not Found")
		}
	}
}
