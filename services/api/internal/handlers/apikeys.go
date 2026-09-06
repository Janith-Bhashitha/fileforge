package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/apikeys"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
)

type APIKeysHandler struct {
	service *apikeys.Service
	audit   *audit.Recorder
}

func NewAPIKeysHandler(service *apikeys.Service, recorder *audit.Recorder) *APIKeysHandler {
	return &APIKeysHandler{service: service, audit: recorder}
}

type apiKeyResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	Revoked    bool       `json:"revoked"`
	// Key is only ever populated on the response to Create - it is not
	// recoverable afterward, only its bcrypt hash is stored.
	Key string `json:"key,omitempty"`
}

func toAPIKeyResponse(k apikeys.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID: k.ID, Name: k.Name, KeyPrefix: k.KeyPrefix,
		CreatedAt: k.CreatedAt, LastUsedAt: k.LastUsedAt, Revoked: k.RevokedAt != nil,
	}
}

type createAPIKeyRequest struct {
	Name string `json:"name"`
}

func (h *APIKeysHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	key, rawKey, err := h.service.Create(r.Context(), claims.UserID, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create api key")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &claims.UserID, Action: "api_key.created",
		ResourceType: "api_key", ResourceID: &key.ID,
		Metadata: map[string]any{"name": key.Name},
	})

	resp := toAPIKeyResponse(*key)
	resp.Key = rawKey
	writeJSON(w, http.StatusCreated, resp)
}

func (h *APIKeysHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	keys, err := h.service.List(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list api keys")
		return
	}

	response := make([]apiKeyResponse, len(keys))
	for i, k := range keys {
		response[i] = toAPIKeyResponse(k)
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *APIKeysHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}

	if err := h.service.Revoke(r.Context(), id, claims.UserID); err != nil {
		if errors.Is(err, apikeys.ErrNotFound) {
			writeError(w, http.StatusNotFound, "api key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to revoke api key")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &claims.UserID, Action: "api_key.revoked",
		ResourceType: "api_key", ResourceID: &id,
	})

	w.WriteHeader(http.StatusNoContent)
}
