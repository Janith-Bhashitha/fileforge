package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	APIPort     string
	DatabaseURL string
	JWTSecret   string
	StorageDir  string
	RedisURL    string

	// Phase 5 hardening knobs. Defaults are generous enough that normal
	// interactive use never notices them, and low enough to blunt abuse.
	RateLimitPerMinute int
	MaxConcurrentJobs  int
	RetentionDays      int

	// Phase 6 storage. STORAGE_BACKEND is "local" (default) or "s3"; the
	// S3 settings are only read when it's "s3". Endpoint/ForcePathStyle
	// exist so MinIO and LocalStack work with the same code path as AWS.
	StorageBackend   string
	S3Bucket         string
	S3Region         string
	S3Endpoint       string
	S3PublicEndpoint string
	S3ForcePathStyle bool
	S3AccessKeyID    string
	S3SecretKey      string
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
		APIPort:            getEnv("API_PORT", "8080"),
		DatabaseURL:        dbURL,
		JWTSecret:          jwtSecret,
		StorageDir:         getEnv("STORAGE_DIR", "./storage"),
		RedisURL:           redisURL,
		RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 120),
		MaxConcurrentJobs:  getEnvInt("MAX_CONCURRENT_JOBS", 20),
		RetentionDays:      getEnvInt("RETENTION_DAYS", 7),
		StorageBackend:     getEnv("STORAGE_BACKEND", "local"),
		S3Bucket:           os.Getenv("S3_BUCKET"),
		S3Region:           getEnv("S3_REGION", "us-east-1"),
		S3Endpoint:         os.Getenv("S3_ENDPOINT"),
		S3PublicEndpoint:   os.Getenv("S3_PUBLIC_ENDPOINT"),
		S3ForcePathStyle:   os.Getenv("S3_FORCE_PATH_STYLE") == "true",
		S3AccessKeyID:      os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretKey:        os.Getenv("S3_SECRET_ACCESS_KEY"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
