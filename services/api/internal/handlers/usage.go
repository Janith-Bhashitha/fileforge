package handlers

import (
	"net/http"
	"time"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/usage"
)

type UsageHandler struct {
	repo *usage.Repository
}

func NewUsageHandler(repo *usage.Repository) *UsageHandler {
	return &UsageHandler{repo: repo}
}

var usageRanges = map[string]time.Duration{
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
}

func (h *UsageHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "30d"
	}
	window, ok := usageRanges[rangeKey]
	if !ok {
		writeError(w, http.StatusBadRequest, "range must be one of 24h, 7d, 30d")
		return
	}

	summary, err := h.repo.Summarize(r.Context(), claims.UserID, time.Now().Add(-window))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load usage")
		return
	}

	writeJSON(w, http.StatusOK, struct {
		Range string `json:"range"`
		*usage.Summary
	}{Range: rangeKey, Summary: summary})
}
