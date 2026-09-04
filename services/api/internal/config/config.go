package config

import (
	"fmt"
	"os"
)

type Config struct {
	APIPort     string
	DatabaseURL string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &Config{
		APIPort:     getEnv("API_PORT", "8080"),
		DatabaseURL: dbURL,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}