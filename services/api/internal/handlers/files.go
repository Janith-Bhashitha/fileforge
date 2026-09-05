package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/validate"
)

type FilesHandler struct {
	repo  *files.Repository
	store storage.Store
}

func NewFilesHandler(repo *files.Repository, store storage.Store) *FilesHandler {
	return &FilesHandler{repo: repo, store: store}
}

type fileResponse struct {
	ID       uuid.UUID `json:"id"`
	Filename string    `json:"filename"`
	MimeType string    `json:"mime_type"`
	Size     int64     `json:"size"`
}

func (h *FilesHandler) Upload(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, validate.MaxUploadSize+1<<20)
	if err := r.ParseMultipartForm(validate.MaxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read upload")
		return
	}

	if err := validate.ValidateSize(int64(len(data))); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	mimeType, err := validate.DetectAndValidate(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	key, err := h.store.Save(r.Context(), data, filepath.Ext(header.Filename))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	f := &files.File{
		ID:         uuid.New(),
		OwnerID:    claims.UserID,
		Filename:   header.Filename,
		MimeType:   mimeType,
		Size:       int64(len(data)),
		Checksum:   checksum,
		StorageKey: key,
	}

	if err := h.repo.Create(r.Context(), f); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file record")
		return
	}

	writeJSON(w, http.StatusCreated, fileResponse{ID: f.ID, Filename: f.Filename, MimeType: f.MimeType, Size: f.Size})
}

func (h *FilesHandler) Download(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid file id")
		return
	}

	f, err := h.repo.GetByID(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, files.ErrNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch file")
		return
	}

	w.Header().Set("Content-Type", f.MimeType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+f.Filename+`"`)
	http.ServeFile(w, r, h.store.LocalPath(f.StorageKey))
}

func (h *FilesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid file id")
		return
	}

	f, err := h.repo.GetByID(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, files.ErrNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch file")
		return
	}

	if err := h.repo.Delete(r.Context(), id, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete file record")
		return
	}

	_ = h.store.Delete(r.Context(), f.StorageKey)

	w.WriteHeader(http.StatusNoContent)
}
