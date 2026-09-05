package handlers

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/batches"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/jobs"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/queue"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/quota"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/validate"
)

type BatchesHandler struct {
	repo      *batches.Repository
	jobsRepo  *jobs.Repository
	filesRepo *files.Repository
	store     storage.Store
	registry  *convert.Registry
	producer  *queue.Producer
	quota     *quota.Tracker
	audit     *audit.Recorder
}

func NewBatchesHandler(repo *batches.Repository, jobsRepo *jobs.Repository, filesRepo *files.Repository, store storage.Store, registry *convert.Registry, producer *queue.Producer, quotaTracker *quota.Tracker, recorder *audit.Recorder) *BatchesHandler {
	return &BatchesHandler{repo: repo, jobsRepo: jobsRepo, filesRepo: filesRepo, store: store, registry: registry, producer: producer, quota: quotaTracker, audit: recorder}
}

type batchResponse struct {
	ID        uuid.UUID `json:"id"`
	Operation string    `json:"operation"`
	Total     int       `json:"total"`
	Completed int       `json:"completed"`
	Failed    int       `json:"failed"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Create accepts multiple files under the "files" form field plus an
// "operation" field, creates one JobItem per file up front (so Total is
// known immediately), and enqueues each independently — a file that fails
// validation/upload/enqueue is counted as failed via the same atomic
// counter the worker uses for processing failures, so Total always stays
// accurate to what the caller submitted.
func (h *BatchesHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, validate.MaxUploadSize*20)
	if err := r.ParseMultipartForm(validate.MaxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}

	operation := r.FormValue("operation")
	version := r.FormValue("version")
	if version == "" {
		version = "v1"
	}

	if _, err := h.registry.Resolve(operation, version); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	options := map[string]string{}
	if raw := r.FormValue("options"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &options); err != nil {
			writeError(w, http.StatusBadRequest, "invalid options: must be a JSON object of strings")
			return
		}
	}

	fileHeaders := r.MultipartForm.File["files"]
	if len(fileHeaders) == 0 {
		writeError(w, http.StatusBadRequest, "no files provided")
		return
	}

	// A batch claims one quota slot per file — otherwise a single 500-file
	// batch would sail past a limit that a 500-job loop would hit.
	allowed, _ := h.quota.Reserve(r.Context(), claims.UserID, len(fileHeaders))
	if !allowed {
		h.audit.Record(r.Context(), audit.Event{
			UserID: &claims.UserID, Action: audit.ActionQuotaExceeded,
			Metadata: map[string]any{"operation": operation, "files": len(fileHeaders), "limit": h.quota.Max()},
		})
		writeError(w, http.StatusTooManyRequests, "batch exceeds how much work you can have in flight at once")
		return
	}

	batch := &batches.Batch{
		ID:        uuid.New(),
		OwnerID:   claims.UserID,
		Operation: operation,
		Options:   options,
		Total:     len(fileHeaders),
		Status:    batches.StatusProcessing,
	}
	if err := h.repo.Create(r.Context(), batch); err != nil {
		_ = h.quota.Release(r.Context(), claims.UserID, len(fileHeaders))
		writeError(w, http.StatusInternalServerError, "failed to create batch")
		return
	}

	failedSoFar := 0
	for _, fh := range fileHeaders {
		if !h.processOneUpload(r, claims.UserID, batch.ID, operation, version, options, fh) {
			_ = h.repo.IncrementFailed(r.Context(), batch.ID)
			failedSoFar++
		}
	}
	batch.Failed = failedSoFar

	// Files rejected before they ever reached a worker will never hit a
	// terminal transition, so their quota slots are handed back here.
	if failedSoFar > 0 {
		_ = h.quota.Release(r.Context(), claims.UserID, failedSoFar)
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &claims.UserID, Action: audit.ActionBatchCreated,
		ResourceType: "batch", ResourceID: &batch.ID,
		Metadata: map[string]any{"operation": operation, "total": batch.Total, "rejected": failedSoFar},
	})

	writeJSON(w, http.StatusCreated, batchResponse{ID: batch.ID, Operation: batch.Operation, Total: batch.Total, Completed: batch.Completed, Failed: batch.Failed, Status: batch.Status, CreatedAt: time.Now()})
}

func (h *BatchesHandler) processOneUpload(r *http.Request, ownerID, batchID uuid.UUID, operation, version string, options map[string]string, fh *multipart.FileHeader) bool {
	file, err := fh.Open()
	if err != nil {
		return false
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return false
	}
	if err := validate.ValidateSize(int64(len(data))); err != nil {
		return false
	}
	mimeType, err := validate.DetectAndValidate(data)
	if err != nil {
		return false
	}

	key, err := h.store.Save(r.Context(), data, filepath.Ext(fh.Filename))
	if err != nil {
		return false
	}

	inputFile := &files.File{
		ID:         uuid.New(),
		OwnerID:    ownerID,
		Filename:   fh.Filename,
		MimeType:   mimeType,
		Size:       int64(len(data)),
		StorageKey: key,
	}
	if err := h.filesRepo.Create(r.Context(), inputFile); err != nil {
		return false
	}

	item := &jobs.JobItem{
		ID:          uuid.New(),
		BatchID:     &batchID,
		InputFileID: inputFile.ID,
		Status:      jobs.StatusQueued,
	}
	if err := h.jobsRepo.CreateItemForBatch(r.Context(), item); err != nil {
		return false
	}

	err = h.producer.Enqueue(r.Context(), queue.Message{
		JobItemID:   item.ID.String(),
		Operation:   operation,
		Version:     version,
		InputFileID: inputFile.ID.String(),
		Options:     options,
	})
	if err != nil {
		errMsg := "failed to enqueue: " + err.Error()
		_ = h.jobsRepo.UpdateItem(r.Context(), item.ID, jobs.StatusFailed, nil, &errMsg, true)
		return false
	}

	return true
}

func (h *BatchesHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch id")
		return
	}

	b, err := h.repo.GetByID(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, batches.ErrNotFound) {
			writeError(w, http.StatusNotFound, "batch not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch batch")
		return
	}

	writeJSON(w, http.StatusOK, batchResponse{ID: b.ID, Operation: b.Operation, Total: b.Total, Completed: b.Completed, Failed: b.Failed, Status: b.Status, CreatedAt: b.CreatedAt})
}

func (h *BatchesHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch id")
		return
	}

	items, err := h.jobsRepo.ListItemsByBatch(r.Context(), id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list batch items")
		return
	}

	response := make([]jobItemResponse, len(items))
	for i, item := range items {
		response[i] = jobItemResponse{
			ID: item.ID, InputFileID: item.InputFileID, OutputFileID: item.OutputFileID,
			Status: item.Status, Attempts: item.Attempts, LastError: item.LastError,
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *BatchesHandler) RetryFailed(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch id")
		return
	}

	b, err := h.repo.GetByID(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, batches.ErrNotFound) {
			writeError(w, http.StatusNotFound, "batch not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch batch")
		return
	}

	items, err := h.jobsRepo.ResetFailedItemsForBatch(r.Context(), id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset failed items")
		return
	}
	if len(items) == 0 {
		writeError(w, http.StatusConflict, "no failed items to retry")
		return
	}

	if err := h.repo.ResetForRetry(r.Context(), id, len(items)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset batch counters")
		return
	}

	for _, item := range items {
		_ = h.producer.Enqueue(r.Context(), queue.Message{
			JobItemID: item.ID.String(), Operation: b.Operation, Version: "v1", InputFileID: item.InputFileID.String(), Options: b.Options,
		})
	}

	writeJSON(w, http.StatusOK, map[string]int{"retried_items": len(items)})
}

// Download streams a ZIP of every completed item's output file — built
// on demand rather than eagerly, since most batches are never downloaded
// as often as they're checked for status.
func (h *BatchesHandler) Download(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch id")
		return
	}

	items, err := h.jobsRepo.ListItemsByBatch(r.Context(), id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list batch items")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="batch-`+id.String()+`.zip"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, item := range items {
		if item.Status != jobs.StatusCompleted || item.OutputFileID == nil {
			continue
		}
		outFile, err := h.filesRepo.GetByIDAny(r.Context(), *item.OutputFileID)
		if err != nil {
			continue
		}
		localPath, release, err := h.store.Fetch(r.Context(), outFile.StorageKey)
		if err != nil {
			continue
		}
		src, err := os.Open(localPath)
		if err != nil {
			release()
			continue
		}
		if zf, err := zw.Create(outFile.Filename); err == nil {
			_, _ = io.Copy(zf, src)
		}
		src.Close()
		release()
	}
}
