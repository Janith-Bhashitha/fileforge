package httpserver

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/batches"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convertsetup"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/handlers"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/jobs"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/metrics"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/queue"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/quota"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/ratelimit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/users"
)

// Deps is what the router needs from main. It's a struct rather than a long
// parameter list because Phase 5 added four more collaborators and positional
// arguments stopped being readable.
type Deps struct {
	Logger    *slog.Logger
	Pool      *pgxpool.Pool
	JWTSecret string
	Store     storage.Store
	Producer  *queue.Producer
	Limiter   *ratelimit.Limiter
	Quota     *quota.Tracker
	Audit     *audit.Recorder
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(d.Logger))
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/healthz", handlers.Health(d.Pool))
	r.Handle("/metrics", promhttp.Handler())

	userRepo := users.NewRepository(d.Pool)
	userService := users.NewService(userRepo)
	authHandler := handlers.NewAuthHandler(userService, d.JWTSecret, d.Audit)

	registry := convertsetup.BuildRegistry()

	fileRepo := files.NewRepository(d.Pool)
	filesHandler := handlers.NewFilesHandler(fileRepo, d.Store, d.Audit)
	convertHandler := handlers.NewConvertHandler(fileRepo, d.Store, registry, d.Audit)

	jobsRepo := jobs.NewRepository(d.Pool)
	jobsHandler := handlers.NewJobsHandler(jobsRepo, fileRepo, registry, d.Producer, d.Quota, d.Audit)

	batchesRepo := batches.NewRepository(d.Pool)
	batchesHandler := handlers.NewBatchesHandler(batchesRepo, jobsRepo, fileRepo, d.Store, registry, d.Producer, d.Quota, d.Audit)

	r.Route("/api/auth", func(r chi.Router) {
		// Rate limiting matters most here — this is the unauthenticated
		// surface where credential stuffing would land.
		r.Use(d.Limiter.Middleware)
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(d.JWTSecret))
			r.Get("/me", authHandler.Me)
		})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(auth.Middleware(d.JWTSecret))
		r.Use(d.Limiter.Middleware)

		r.Get("/files", filesHandler.List)
		r.Post("/files", filesHandler.Upload)
		r.Get("/files/{id}/download", filesHandler.Download)
		r.Delete("/files/{id}", filesHandler.Delete)

		r.Post("/convert", convertHandler.Convert)

		// The presigned upload path only exists on the S3 backend; on local
		// storage there is nothing to presign and POST /files stays the
		// only way in. Registering it conditionally means a caller gets a
		// clean 404 rather than an endpoint that half-works.
		if s3Store, ok := d.Store.(*storage.S3Store); ok {
			presignHandler := handlers.NewPresignHandler(fileRepo, s3Store, d.Audit)
			r.Post("/files/presign", presignHandler.Presign)
			r.Post("/files/presign/complete", presignHandler.Complete)
		}

		r.Post("/jobs", jobsHandler.Create)
		r.Get("/jobs/{id}", jobsHandler.Get)
		r.Get("/jobs/{id}/items", jobsHandler.ListItems)
		r.Post("/jobs/{id}/cancel", jobsHandler.Cancel)
		r.Post("/jobs/{id}/retry", jobsHandler.Retry)

		r.Post("/batches", batchesHandler.Create)
		r.Get("/batches/{id}", batchesHandler.Get)
		r.Get("/batches/{id}/items", batchesHandler.ListItems)
		r.Post("/batches/{id}/retry-failed", batchesHandler.RetryFailed)
		r.Get("/batches/{id}/download", batchesHandler.Download)
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

			elapsed := time.Since(start)

			// Label by chi's route pattern, not the raw path: "/api/v1/files/{id}"
			// stays one time series instead of one per file ID.
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			metrics.HTTPRequests.WithLabelValues(r.Method, route, strconv.Itoa(ww.Status())).Inc()
			metrics.HTTPDuration.WithLabelValues(r.Method, route).Observe(elapsed.Seconds())

			logger.Info("request",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", elapsed.Milliseconds(),
			)
		})
	}
}
