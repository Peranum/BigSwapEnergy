// Package main starts the BigSwapEnergy HTTP service.
//
// It wires configuration, logging, Ethereum RPC client, and HTTP handlers
// to expose a GET /estimate endpoint for Uniswap V2 swap estimations.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"bigswapenergy/internal/infrastructure/ethereum"
	"bigswapenergy/internal/infrastructure/uniswap_v2"
	"bigswapenergy/internal/presentation/http"
	"bigswapenergy/internal/shared/config"
	"bigswapenergy/internal/shared/logger"
	estimate "bigswapenergy/internal/usecases"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// main is the entrypoint that invokes run and exits with a non-zero status
// code on error.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run initializes configuration, logging, network clients and HTTP routes,
// then runs the server until a shutdown signal is received.
func run() error {
	log := logger.NewLogger()
	defer log.Sync()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ethClient, err := ethereum.NewEthereumClient(cfg.Blockchain.EthereumRPCURL, cfg.Blockchain.ConnectionPoolSize, log)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum connection pool: %w", err)
	}
	defer ethClient.Close()

	uniswapV2Client := uniswap_v2.NewUniswapV2Client(ethClient, log)
	estimateService := estimate.NewEstimateService(uniswapV2Client, log)
	estimateHandler := http.NewEstimateHandler(estimateService, log, cfg)

	handler := http.ApplyMiddleware(
		estimateHandler.EstimateSwapAmount,
		log,
		estimateHandler,
	)

	server := &fasthttp.Server{
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("Starting server", zap.String("address", cfg.Server.Address))
		errCh <- server.ListenAndServe(cfg.Server.Address)
	}()

	select {
	case <-ctx.Done():
		log.Info("Received shutdown signal, starting graceful shutdown")
	case err := <-errCh:
		if err != nil {
			log.Error("Server error occurred", zap.Error(err))
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("Error during server shutdown", zap.Error(err))
	} else {
		log.Info("Server shutdown completed successfully")
	}

	return nil
}
