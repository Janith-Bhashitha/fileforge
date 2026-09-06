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

	// AI features. OCR and Document Insights need no key; only ai-analyze
	// does. Left empty, ai-analyze fails clearly per-request rather than the
	// app refusing to start - the rest of FileForge doesn't depend on it.
	GeminiAPIKey string
	GeminiModel  string

	// Password reset email. Left unconfigured, RequestPasswordReset still
	// works but the handler logs the reset link instead of emailing it -
	// same "degrade, don't break" pattern as GeminiAPIKey.
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// Where the frontend actually lives, so a password-reset email can link
	// to a real page (http://localhost:5173/reset-password?token=... in
	// dev, the deployed origin in production) instead of guessing.
	FrontendURL string
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
		GeminiAPIKey:       os.Getenv("GEMINI_API_KEY"),
		GeminiModel:        getEnv("GEMINI_MODEL", "gemini-flash-lite-latest"),
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           getEnv("SMTP_PORT", "587"),
		SMTPUsername:       os.Getenv("SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:           getEnv("SMTP_FROM", os.Getenv("SMTP_USERNAME")),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
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
