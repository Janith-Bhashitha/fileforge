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
	RedisURL    string
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

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}

	return &Config{
		APIPort:     getEnv("API_PORT", "8080"),
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
		StorageDir:  getEnv("STORAGE_DIR", "./storage"),
		RedisURL:    redisURL,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
