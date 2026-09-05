package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/jobs"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/queue"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/quota"
)

type JobsHandler struct {
	repo      *jobs.Repository
	filesRepo *files.Repository
	registry  *convert.Registry
	producer  *queue.Producer
	quota     *quota.Tracker
	audit     *audit.Recorder
}

func NewJobsHandler(repo *jobs.Repository, filesRepo *files.Repository, registry *convert.Registry, producer *queue.Producer, quotaTracker *quota.Tracker, recorder *audit.Recorder) *JobsHandler {
	return &JobsHandler{repo: repo, filesRepo: filesRepo, registry: registry, producer: producer, quota: quotaTracker, audit: recorder}
}

type createJobRequest struct {
	FileID    string            `json:"file_id"`
	Operation string            `json:"operation"`
	Version   string            `json:"version"`
	Options   map[string]string `json:"options"`
}

type jobResponse struct {
	ID        uuid.UUID `json:"id"`
	Operation string    `json:"operation"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	Error     *string   `json:"error,omitempty"`
}

type jobItemResponse struct {
	ID           uuid.UUID  `json:"id"`
	InputFileID  uuid.UUID  `json:"input_file_id"`
	OutputFileID *uuid.UUID `json:"output_file_id,omitempty"`
	Status       string     `json:"status"`
	Attempts     int        `json:"attempts"`
	LastError    *string    `json:"last_error,omitempty"`
}

// Create validates the file and operation belong to / exist for this
// request, writes the Job+JobItem row, and enqueues the work — in that
// order, so a job is never visible to the owner before it's actually
// queued, and never queued without a persisted row backing it.
func (h *JobsHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Version == "" {
		req.Version = "v1"
	}

	fileID, err := uuid.Parse(req.FileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid file_id")
		return
	}

	inputFile, err := h.filesRepo.GetByID(r.Context(), fileID, claims.UserID)
	if err != nil {
		if errors.Is(err, files.ErrNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch file")
		return
	}

	if _, err := h.registry.Resolve(req.Operation, req.Version); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Quota is claimed before the row is written, so a rejected submission
	// leaves nothing behind to clean up.
	allowed, _ := h.quota.Reserve(r.Context(), claims.UserID, 1)
	if !allowed {
		h.audit.Record(r.Context(), audit.Event{
			UserID: &claims.UserID, Action: audit.ActionQuotaExceeded,
			Metadata: map[string]any{"operation": req.Operation, "limit": h.quota.Max()},
		})
		writeError(w, http.StatusTooManyRequests, "too many jobs in flight, wait for some to finish")
		return
	}

	job := &jobs.Job{
		ID:        uuid.New(),
		OwnerID:   claims.UserID,
		Operation: req.Operation,
		Status:    jobs.StatusQueued,
	}
	item := &jobs.JobItem{
		ID:          uuid.New(),
		JobID:       &job.ID,
		InputFileID: inputFile.ID,
		Status:      jobs.StatusQueued,
	}

	if err := h.repo.CreateWithItem(r.Context(), job, item); err != nil {
		_ = h.quota.Release(r.Context(), claims.UserID, 1)
		writeError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	err = h.producer.Enqueue(r.Context(), queue.Message{
		JobItemID:   item.ID.String(),
		Operation:   req.Operation,
		Version:     req.Version,
		InputFileID: inputFile.ID.String(),
		Options:     req.Options,
	})
	if err != nil {
		// The row exists but nothing will ever pick it up - mark it FAILED
		// immediately rather than leaving a job stuck at "queued" forever.
		errMsg := "failed to enqueue: " + err.Error()
		_ = h.repo.UpdateJobStatus(r.Context(), job.ID, jobs.StatusFailed, &errMsg)
		_ = h.quota.Release(r.Context(), claims.UserID, 1)
		writeError(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &claims.UserID, Action: audit.ActionJobCreated,
		ResourceType: "job", ResourceID: &job.ID,
		Metadata: map[string]any{"operation": req.Operation, "input_file_id": inputFile.ID.String()},
	})

	writeJSON(w, http.StatusCreated, jobResponse{ID: job.ID, Operation: job.Operation, Status: job.Status, Progress: job.Progress})
}

func (h *JobsHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.repo.GetJob(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch job")
		return
	}

	writeJSON(w, http.StatusOK, jobResponse{ID: job.ID, Operation: job.Operation, Status: job.Status, Progress: job.Progress, Error: job.Error})
}

func (h *JobsHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	items, err := h.repo.ListItemsByJob(r.Context(), id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list job items")
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

func (h *JobsHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	cancelled, err := h.repo.CancelIfQueued(r.Context(), id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel job")
		return
	}
	if !cancelled {
		writeError(w, http.StatusConflict, "job is no longer cancellable (already processing or finished)")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": jobs.StatusCancelled})
}

func (h *JobsHandler) Retry(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.repo.GetJob(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch job")
		return
	}

	items, err := h.repo.ResetFailedItemsForRetry(r.Context(), id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset failed items")
		return
	}
	if len(items) == 0 {
		writeError(w, http.StatusConflict, "no failed items to retry")
		return
	}

	for _, item := range items {
		_ = h.producer.Enqueue(r.Context(), queue.Message{
			JobItemID:   item.ID.String(),
			Operation:   job.Operation,
			Version:     "v1",
			InputFileID: item.InputFileID.String(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]int{"retried_items": len(items)})
}
