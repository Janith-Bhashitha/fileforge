// Package workerhealth gives every worker binary the same minimal
// liveness and metrics endpoints, so Docker (or Prometheus) can reach a
// worker without each worker's main.go repeating the same handlers.
package workerhealth

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Serve(logger *slog.Logger, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("healthz server error", "error", err)
	}
}
