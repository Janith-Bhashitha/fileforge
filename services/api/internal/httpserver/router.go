package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/imageops"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/office"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/pdfops"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert/txtops"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/handlers"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/users"
)

func NewRouter(logger *slog.Logger, pool *pgxpool.Pool, jwtSecret string, store storage.Store) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/healthz", handlers.Health(pool))

	userRepo := users.NewRepository(pool)
	userService := users.NewService(userRepo)
	authHandler := handlers.NewAuthHandler(userService, jwtSecret)

	fileRepo := files.NewRepository(pool)
	filesHandler := handlers.NewFilesHandler(fileRepo, store)
	convertHandler := handlers.NewConvertHandler(fileRepo, store, buildRegistry())

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
	})

	return r
}

// buildRegistry registers every Phase 2 operation under its "name:version"
// key. New operations (Phase 7+ AI-backed ones included) register here too,
// reusing the same Processor interface.
func buildRegistry() *convert.Registry {
	reg := convert.NewRegistry()
	reg.Register("image-to-pdf", "v1", imageops.ImageToPDFProcessor{})
	reg.Register("pdf-to-image", "v1", imageops.PDFToImageProcessor{})
	reg.Register("image-convert", "v1", imageops.ImageConvertProcessor{})
	reg.Register("image-resize", "v1", imageops.ImageResizeProcessor{})
	reg.Register("docx-to-pdf", "v1", office.OfficeToPDFProcessor{})
	reg.Register("pptx-to-pdf", "v1", office.OfficeToPDFProcessor{})
	reg.Register("xlsx-to-pdf", "v1", office.OfficeToPDFProcessor{})
	reg.Register("txt-to-pdf", "v1", txtops.TxtToPDFProcessor{})
	reg.Register("pdf-merge", "v1", pdfops.MergeProcessor{})
	reg.Register("pdf-split", "v1", pdfops.SplitProcessor{})
	reg.Register("pdf-compress", "v1", pdfops.CompressProcessor{})
	return reg
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
