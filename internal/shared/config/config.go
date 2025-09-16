package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig
	Blockchain BlockchainConfig
	RateLimit  RateLimitConfig
}

type ServerConfig struct {
	Address         string        `yaml:"address"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type BlockchainConfig struct {
	EthereumRPCURL     string `yaml:"ethereum_rpc_url"`
	ConnectionPoolSize int    `yaml:"connection_pool_size"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
}

func LoadConfig(configPath string) (*Config, error) {
	config := getDefaultConfig()

	if configPath != "" {
		if err := loadFromYAML(configPath, config); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	rpcURL := os.Getenv("ETHEREUM_RPC_URL")
	if rpcURL == "" {
		return nil, fmt.Errorf("ETHEREUM_RPC_URL environment variable is required")
	}
	config.Blockchain.EthereumRPCURL = rpcURL

	return config, nil
}

func loadFromYAML(configPath string, config *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, config)
}

func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:         ":1337",
			ShutdownTimeout: 30 * time.Second,
		},
		Blockchain: BlockchainConfig{
			ConnectionPoolSize: 5,
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 600, // 10 requests per second (Infura-friendly)
		},
	}
}
