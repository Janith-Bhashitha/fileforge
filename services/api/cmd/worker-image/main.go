package main

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/batches"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/config"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convertsetup"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/db"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/jobs"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/logging"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/queue"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/quota"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/workerconsumer"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/workerhealth"
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

	store, err := storage.NewLocalStore(cfg.StorageDir)
	if err != nil {
		logger.Error("failed to init storage", "error", err)
		return
	}

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("failed to parse redis url", "error", err)
		return
	}

	producer, err := queue.NewProducer(cfg.RedisURL)
	if err != nil {
		logger.Error("failed to init queue producer", "error", err)
		return
	}

	quotaTracker, err := quota.New(cfg.RedisURL, cfg.MaxConcurrentJobs)
	if err != nil {
		logger.Error("failed to init quota tracker", "error", err)
		return
	}

	hostname, _ := os.Hostname()

	runner := &workerconsumer.Runner{
		Logger:      logger,
		Redis:       redis.NewClient(redisOpts),
		Producer:    producer,
		Stream:      "stream:image",
		Group:       "cg:worker-image",
		Consumer:    hostname,
		Registry:    convertsetup.BuildRegistry(),
		JobsRepo:    jobs.NewRepository(pool),
		BatchesRepo: batches.NewRepository(pool),
		FilesRepo:   files.NewRepository(pool),
		Store:       store,
		Quota:       quotaTracker,
	}

	go workerhealth.Serve(logger, ":8080")

	if err := runner.Run(ctx); err != nil {
		logger.Error("worker stopped", "error", err)
	}
}
