package config

import "os"

type Config struct {
	APIPort string
}

func Load() (*Config, error) {
	return &Config{
		APIPort: getEnv("API_PORT", "8080"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}