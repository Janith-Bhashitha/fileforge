package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Health(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","db":"unreachable"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","db":"ok"}`))
	}
}