package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/validate"
)

// PresignHandler implements the Phase 6 upload path: the browser asks for a
// short-lived URL, PUTs the bytes straight to object storage, then tells the
// API the upload landed. File bytes never pass through the API at all, which
// is what makes it cheap to run several small API replicas.
//
// Only registered when the S3 backend is active — on local storage there is
// nothing to presign, and the multipart POST /files path stays the way in.
type PresignHandler struct {
	repo  *files.Repository
	store *storage.S3Store
	audit *audit.Recorder
}

func NewPresignHandler(repo *files.Repository, store *storage.S3Store, recorder *audit.Recorder) *PresignHandler {
	return &PresignHandler{repo: repo, store: store, audit: recorder}
}

const presignTTL = 10 * time.Minute

type presignRequest struct {
	Filename string `json:"filename"`
}

type presignResponse struct {
	UploadURL  string    `json:"upload_url"`
	StorageKey string    `json:"storage_key"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (h *PresignHandler) Presign(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.ClaimsFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req presignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Filename == "" {
		writeError(w, http.StatusBadRequest, "filename is required")
		return
	}

	// Only the extension is taken from the caller's filename, and even that
	// goes into a UUID-based key — the name itself never becomes part of the
	// object key, so it can't be used to traverse or enumerate.
	url, key, err := h.store.PresignPut(r.Context(), filepath.Ext(req.Filename), presignTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to presign upload")
		return
	}

	writeJSON(w, http.StatusOK, presignResponse{
		UploadURL:  url,
		StorageKey: key,
		ExpiresAt:  time.Now().Add(presignTTL),
	})
}

type completeRequest struct {
	StorageKey string `json:"storage_key"`
	Filename   string `json:"filename"`
	MimeType   string `json:"mime_type"`
	Size       int64  `json:"size"`
}

// Complete registers a presigned upload as a real file. The browser saying
// "it worked" is not evidence, so the object's existence is confirmed
// against storage before any row is written — otherwise a caller could
// register file records for objects that were never uploaded.
func (h *PresignHandler) Complete(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req completeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.StorageKey == "" || req.Filename == "" {
		writeError(w, http.StatusBadRequest, "storage_key and filename are required")
		return
	}
	if err := validate.ValidateSize(req.Size); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}

	exists, err := h.store.Exists(r.Context(), req.StorageKey)
	if err != nil || !exists {
		writeError(w, http.StatusBadRequest, "no uploaded object found for that storage key")
		return
	}

	// The uploaded bytes are sniffed rather than trusting the client's
	// declared mime type — same rule as the multipart path, just applied
	// after the fact since the API never saw the upload.
	localPath, cleanup, err := h.store.Fetch(r.Context(), req.StorageKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read uploaded object")
		return
	}
	defer cleanup()

	mimeType, err := validate.DetectFile(localPath)
	if err != nil {
		_ = h.store.Delete(r.Context(), req.StorageKey)
		writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}

	f := &files.File{
		ID:         uuid.New(),
		OwnerID:    claims.UserID,
		Filename:   filepath.Base(req.Filename),
		MimeType:   mimeType,
		Size:       req.Size,
		StorageKey: req.StorageKey,
	}
	if err := h.repo.Create(r.Context(), f); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file record")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &claims.UserID, Action: audit.ActionFileUploaded,
		ResourceType: "file", ResourceID: &f.ID,
		Metadata: map[string]any{"filename": f.Filename, "via": "presigned"},
	})

	writeJSON(w, http.StatusCreated, fileResponse{ID: f.ID, Filename: f.Filename, MimeType: f.MimeType, Size: f.Size})
}
