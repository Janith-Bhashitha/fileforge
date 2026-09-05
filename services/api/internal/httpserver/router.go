package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convertsetup"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/handlers"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/jobs"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/queue"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/users"
)

func NewRouter(logger *slog.Logger, pool *pgxpool.Pool, jwtSecret string, store storage.Store, producer *queue.Producer) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/healthz", handlers.Health(pool))

	userRepo := users.NewRepository(pool)
	userService := users.NewService(userRepo)
	authHandler := handlers.NewAuthHandler(userService, jwtSecret)

	registry := convertsetup.BuildRegistry()

	fileRepo := files.NewRepository(pool)
	filesHandler := handlers.NewFilesHandler(fileRepo, store)
	convertHandler := handlers.NewConvertHandler(fileRepo, store, registry)

	jobsRepo := jobs.NewRepository(pool)
	jobsHandler := handlers.NewJobsHandler(jobsRepo, fileRepo, registry, producer)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(jwtSecret))
			r.Get("/me", authHandler.Me)
		})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(auth.Middleware(jwtSecret))

		r.Get("/files", filesHandler.List)
		r.Post("/files", filesHandler.Upload)
		r.Get("/files/{id}/download", filesHandler.Download)
		r.Delete("/files/{id}", filesHandler.Delete)

		r.Post("/convert", convertHandler.Convert)

		r.Post("/jobs", jobsHandler.Create)
		r.Get("/jobs/{id}", jobsHandler.Get)
		r.Get("/jobs/{id}/items", jobsHandler.ListItems)
		r.Post("/jobs/{id}/cancel", jobsHandler.Cancel)
		r.Post("/jobs/{id}/retry", jobsHandler.Retry)
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
