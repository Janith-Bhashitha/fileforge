package main

import (
	"context"
	"net/http"
	"time"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/config"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/db"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/httpserver"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/logging"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/metrics"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/queue"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/quota"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/ratelimit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
)

func main() {
	logger := logging.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		return
	}
	defer pool.Close()

	store, err := storage.New(ctx, storage.Settings{
		Backend:        cfg.StorageBackend,
		Dir:            cfg.StorageDir,
		Bucket:         cfg.S3Bucket,
		Region:         cfg.S3Region,
		Endpoint:       cfg.S3Endpoint,
		PublicEndpoint: cfg.S3PublicEndpoint,
		ForcePathStyle: cfg.S3ForcePathStyle,
		AccessKeyID:    cfg.S3AccessKeyID,
		SecretKey:      cfg.S3SecretKey,
	})
	if err != nil {
		logger.Error("failed to init storage", "error", err)
		return
	}

	producer, err := queue.NewProducer(cfg.RedisURL)
	if err != nil {
		logger.Error("failed to init queue producer", "error", err)
		return
	}

	limiter, err := ratelimit.New(cfg.RedisURL, cfg.RateLimitPerMinute, time.Minute)
	if err != nil {
		logger.Error("failed to init rate limiter", "error", err)
		return
	}

	quotaTracker, err := quota.New(cfg.RedisURL, cfg.MaxConcurrentJobs)
	if err != nil {
		logger.Error("failed to init quota tracker", "error", err)
		return
	}

	go metrics.WatchQueueDepth(ctx, logger, cfg.RedisURL,
		[]string{"stream:pdf", "stream:image", "stream:office"}, 15*time.Second)

	router := httpserver.NewRouter(httpserver.Deps{
		Logger:    logger,
		Pool:      pool,
		JWTSecret: cfg.JWTSecret,
		Store:     store,
		Producer:  producer,
		Limiter:   limiter,
		Quota:     quotaTracker,
		Audit:     audit.NewRecorder(pool, logger),
	})

	logger.Info("starting server",
		"port", cfg.APIPort,
		"storage_dir", cfg.StorageDir,
		"rate_limit_per_minute", cfg.RateLimitPerMinute,
		"max_concurrent_jobs", cfg.MaxConcurrentJobs,
	)
	if err := http.ListenAndServe(":"+cfg.APIPort, router); err != nil {
		logger.Error("server error", "error", err)
	}
}
