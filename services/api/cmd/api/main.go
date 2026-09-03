package main

import (
	"net/http"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/config"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/httpserver"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/logging"
)

func main() {
	logger := logging.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return
	}

	router := httpserver.NewRouter(logger)

	logger.Info("starting server", "port", cfg.APIPort)
	if err := http.ListenAndServe(":"+cfg.APIPort, router); err != nil {
		logger.Error("server error", "error", err)
	}
}