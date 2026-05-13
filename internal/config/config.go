package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv        string
	Port          string
	DatabaseURL   string
	RedisURL      string
	EVMRPCURL     string
	ChainsPath    string
	MarketURL     string
	MarketAPIKey  string
	AlchemyAPIKey string

	RequestTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           getEnv("PORT", "8502"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://web3:web3@localhost:5432/web3?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/0"),
		EVMRPCURL:      os.Getenv("EVM_RPC_URL"),
		ChainsPath:     getEnv("CHAINS_JSON_PATH", "seeds/chains.json"),
		MarketURL:      getEnv("MARKET_DATA_URL", "https://api.coingecko.com/api/v3"),
		MarketAPIKey:   os.Getenv("MARKET_DATA_API_KEY"),
		AlchemyAPIKey:  os.Getenv("ALCHEMY_API_KEY"),
		RequestTimeout: getDurationEnv("REQUEST_TIMEOUT_SECONDS", 15) * time.Second,
	}

	if strings.TrimSpace(cfg.Port) == "" {
		return Config{}, errors.New("PORT is required")
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if strings.TrimSpace(cfg.RedisURL) == "" {
		return Config{}, errors.New("REDIS_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func getDurationEnv(key string, fallbackSeconds int) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return time.Duration(fallbackSeconds)
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Duration(fallbackSeconds)
	}

	return time.Duration(seconds)
}
