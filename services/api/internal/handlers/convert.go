package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/metrics"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
)

type ConvertHandler struct {
	repo     *files.Repository
	store    storage.Store
	registry *convert.Registry
	audit    *audit.Recorder
}

func NewConvertHandler(repo *files.Repository, store storage.Store, registry *convert.Registry, recorder *audit.Recorder) *ConvertHandler {
	return &ConvertHandler{repo: repo, store: store, registry: registry, audit: recorder}
}

type convertRequest struct {
	FileID    string            `json:"file_id"`
	FileIDs   []string          `json:"file_ids"`
	Operation string            `json:"operation"`
	Version   string            `json:"version"`
	Options   map[string]string `json:"options"`
}

// Convert runs a registered operation synchronously (Phase 2 has no queue
// yet — Phase 3 wraps this same registry/processor call in a worker).
//
// Most operations take one file (file_id). Multi-input operations like
// pdf-merge accept file_ids instead, listed in the order they should be
// combined; every file is re-verified as owned by the caller before its
// local path is ever touched.
func (h *ConvertHandler) Convert(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req convertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Version == "" {
		req.Version = "v1"
	}

	rawIDs := req.FileIDs
	if len(rawIDs) == 0 {
		rawIDs = []string{req.FileID}
	}

	var inputFiles []*files.File
	var inputPaths []string
	for _, raw := range rawIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid file id: "+raw)
			return
		}
		f, err := h.repo.GetByID(r.Context(), id, claims.UserID)
		if err != nil {
			if errors.Is(err, files.ErrNotFound) {
				writeError(w, http.StatusNotFound, "file not found: "+raw)
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to fetch file")
			return
		}
		localPath, release, fetchErr := h.store.Fetch(r.Context(), f.StorageKey)
		if fetchErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch file")
			return
		}
		defer release()

		inputFiles = append(inputFiles, f)
		inputPaths = append(inputPaths, localPath)
	}

	processor, err := h.registry.Resolve(req.Operation, req.Version)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	options := req.Options
	if options == nil {
		options = map[string]string{}
	}
	if len(inputPaths) > 1 {
		options["inputs"] = strings.Join(inputPaths, ";")
	}

	started := time.Now()
	result, err := processor.Process(r.Context(), convert.ConversionRequest{
		InputPath: inputPaths[0],
		Options:   options,
	})
	metrics.ConversionDuration.WithLabelValues(req.Operation).Observe(time.Since(started).Seconds())
	if err != nil {
		metrics.ConversionsTotal.WithLabelValues(req.Operation, "error").Inc()
		writeError(w, http.StatusUnprocessableEntity, "conversion failed: "+err.Error())
		return
	}
	metrics.ConversionsTotal.WithLabelValues(req.Operation, "success").Inc()

	var size int64
	if info, statErr := os.Stat(result.OutputPath); statErr == nil {
		size = info.Size()
	}

	outputKey, err := h.store.SaveFile(r.Context(), result.OutputPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store output file")
		return
	}

	// Display name is derived from the original upload, never from the
	// processor's internal (often UUID-based) output path.
	displayName := convert.DisplayFilename(inputFiles[0].Filename, req.Operation, filepath.Ext(result.OutputPath))

	outputFile := &files.File{
		ID:                uuid.New(),
		OwnerID:           claims.UserID,
		Filename:          displayName,
		MimeType:          result.MimeType,
		Size:              size,
		Checksum:          "",
		StorageKey:        outputKey,
		DerivedFromFileID: &inputFiles[0].ID,
		Operation:         &req.Operation,
	}

	if err := h.repo.Create(r.Context(), outputFile); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save output file record")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &claims.UserID, Action: audit.ActionFileConverted,
		ResourceType: "file", ResourceID: &outputFile.ID,
		Metadata: map[string]any{"operation": req.Operation, "input_file_id": inputFiles[0].ID.String()},
	})

	writeJSON(w, http.StatusOK, fileResponse{
		ID:       outputFile.ID,
		Filename: outputFile.Filename,
		MimeType: outputFile.MimeType,
		Size:     outputFile.Size,
	})
}
