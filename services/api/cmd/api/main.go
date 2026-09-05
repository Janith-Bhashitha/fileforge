package main

import (
	"context"
	"net/http"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/config"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/db"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/httpserver"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/logging"
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

	store, err := storage.NewLocalStore(cfg.StorageDir)
	if err != nil {
		logger.Error("failed to init storage", "error", err)
		return
	}

	router := httpserver.NewRouter(logger, pool, cfg.JWTSecret, store)

	logger.Info("starting server", "port", cfg.APIPort, "storage_dir", cfg.StorageDir)
	if err := http.ListenAndServe(":"+cfg.APIPort, router); err != nil {
		logger.Error("server error", "error", err)
	}
}
