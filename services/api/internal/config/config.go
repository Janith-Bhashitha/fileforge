package config

import (
	"fmt"
	"os"
)

type Config struct {
	APIPort     string
	DatabaseURL string
	JWTSecret   string
	StorageDir  string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return &Config{
		APIPort:     getEnv("API_PORT", "8080"),
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
		StorageDir:  getEnv("STORAGE_DIR", "./storage"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
