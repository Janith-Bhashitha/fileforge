// Package workerhealth gives every worker binary the same minimal
// liveness endpoint, so Docker (or anything else) can check whether the
// process is up without each worker's main.go repeating the same handler.
package workerhealth

import (
	"log/slog"
	"net/http"
)

func Serve(logger *slog.Logger, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("healthz server error", "error", err)
	}
}
