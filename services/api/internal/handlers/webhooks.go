package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/audit"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/webhooks"
)

type WebhooksHandler struct {
	repo  *webhooks.Repository
	dis   *webhooks.Dispatcher
	audit *audit.Recorder
}

func NewWebhooksHandler(repo *webhooks.Repository, dispatcher *webhooks.Dispatcher, recorder *audit.Recorder) *WebhooksHandler {
	return &WebhooksHandler{repo: repo, dis: dispatcher, audit: recorder}
}

type webhookResponse struct {
	ID         uuid.UUID `json:"id"`
	URL        string    `json:"url"`
	EventTypes []string  `json:"event_types"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	// Secret is only ever returned on Create - after that only the receiver
	// (who was given it once) and the database have it.
	Secret string `json:"secret,omitempty"`
}

func toWebhookResponse(w webhooks.Webhook) webhookResponse {
	return webhookResponse{ID: w.ID, URL: w.URL, EventTypes: w.EventTypes, Active: w.Active, CreatedAt: w.CreatedAt}
}

type createWebhookRequest struct {
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types"`
}

// deliveryResponse mirrors webhooks.Delivery for the wire. The domain struct
// carries no json tags, and the raw Payload is of no use to the dashboard -
// it would only ship a base64 blob per row - so the log view gets the
// delivery metadata and nothing else.
type deliveryResponse struct {
	ID              uuid.UUID  `json:"id"`
	WebhookID       uuid.UUID  `json:"webhook_id"`
	EventType       string     `json:"event_type"`
	Status          string     `json:"status"`
	Attempts        int        `json:"attempts"`
	ResponseStatus  *int       `json:"response_status,omitempty"`
	LastAttemptedAt *time.Time `json:"last_attempted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func toDeliveryResponse(d webhooks.Delivery) deliveryResponse {
	return deliveryResponse{
		ID: d.ID, WebhookID: d.WebhookID, EventType: d.EventType,
		Status: d.Status, Attempts: d.Attempts, ResponseStatus: d.ResponseStatus,
		LastAttemptedAt: d.LastAttemptedAt, CreatedAt: d.CreatedAt,
	}
}

func (h *WebhooksHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	parsed, err := url.Parse(req.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		writeError(w, http.StatusBadRequest, "url must be a valid http(s) URL")
		return
	}
	if len(req.EventTypes) == 0 {
		writeError(w, http.StatusBadRequest, "at least one event type is required")
		return
	}
	for _, et := range req.EventTypes {
		if !isValidEventType(et) {
			writeError(w, http.StatusBadRequest, "unknown event type: "+et)
			return
		}
	}

	secretBytes := make([]byte, 24)
	if _, err := rand.Read(secretBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate webhook secret")
		return
	}

	hook := &webhooks.Webhook{
		ID:         uuid.New(),
		OwnerID:    claims.UserID,
		URL:        req.URL,
		EventTypes: req.EventTypes,
		Secret:     hex.EncodeToString(secretBytes),
		Active:     true,
	}
	if err := h.repo.Create(r.Context(), hook); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create webhook")
		return
	}

	h.audit.Record(r.Context(), audit.Event{
		UserID: &claims.UserID, Action: "webhook.created",
		ResourceType: "webhook", ResourceID: &hook.ID,
		Metadata: map[string]any{"url": hook.URL, "event_types": hook.EventTypes},
	})

	resp := toWebhookResponse(*hook)
	resp.Secret = hook.Secret
	writeJSON(w, http.StatusCreated, resp)
}

func isValidEventType(et string) bool {
	for _, valid := range webhooks.AllEventTypes {
		if et == valid {
			return true
		}
	}
	return false
}

func (h *WebhooksHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	hooks, err := h.repo.ListByOwner(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list webhooks")
		return
	}

	response := make([]webhookResponse, len(hooks))
	for i, hk := range hooks {
		response[i] = toWebhookResponse(hk)
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *WebhooksHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook id")
		return
	}

	if err := h.repo.Delete(r.Context(), id, claims.UserID); err != nil {
		if errors.Is(err, webhooks.ErrNotFound) {
			writeError(w, http.StatusNotFound, "webhook not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete webhook")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WebhooksHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook id")
		return
	}

	deliveries, err := h.repo.ListDeliveries(r.Context(), id, claims.UserID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list deliveries")
		return
	}

	response := make([]deliveryResponse, len(deliveries))
	for i, d := range deliveries {
		response[i] = toDeliveryResponse(d)
	}
	writeJSON(w, http.StatusOK, response)
}

// SendTest fires a synthetic event at the webhook immediately, so a user
// can confirm their endpoint and secret verification actually work without
// needing to wait for (or fake) a real job completing.
func (h *WebhooksHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook id")
		return
	}

	hook, err := h.repo.GetByID(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, webhooks.ErrNotFound) {
			writeError(w, http.StatusNotFound, "webhook not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch webhook")
		return
	}

	// Create rejects an empty event list, so this only guards a row edited
	// directly in the database - but indexing [0] blind would panic the
	// handler rather than fail the request, which is never the right trade.
	if len(hook.EventTypes) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "webhook has no event types configured")
		return
	}

	h.dis.Send(r.Context(), claims.UserID, hook.EventTypes[0], map[string]string{
		"message": "This is a test delivery from FileForge.",
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "test delivery queued"})
}
