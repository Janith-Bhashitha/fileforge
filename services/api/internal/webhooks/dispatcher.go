package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Dispatcher sends webhook deliveries in-process rather than through a
// separate queue or service. At the volume a single-user (or small-team)
// deployment like this one generates, a dedicated delivery service would
// be solving a scale problem that doesn't exist yet; if delivery volume
// ever justified it, this is the seam where it would move out.
type Dispatcher struct {
	repo       *Repository
	logger     *slog.Logger
	httpClient *http.Client
}

func NewDispatcher(repo *Repository, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		repo:   repo,
		logger: logger,
		// A slow or hung receiver must not be able to stall the caller (a
		// worker mid-job) beyond a few seconds.
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type eventPayload struct {
	Event     string          `json:"event"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

const maxDeliveryAttempts = 3

// Send looks up this owner's active webhooks for eventType and delivers to
// each independently in its own goroutine, so one slow or dead endpoint
// never delays another, or the caller. Best-effort by design: a delivery
// failure is retried and logged, never propagated back to whatever
// triggered the event (a job completing must never fail because someone's
// webhook receiver is down).
func (d *Dispatcher) Send(ctx context.Context, ownerID uuid.UUID, eventType string, data any) {
	hooks, err := d.repo.ListActiveForOwnerAndEvent(ctx, ownerID, eventType)
	if err != nil {
		d.logger.Error("failed to list webhooks for dispatch", "error", err, "event", eventType)
		return
	}
	if len(hooks) == 0 {
		return
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		d.logger.Error("failed to marshal webhook payload", "error", err, "event", eventType)
		return
	}

	body, err := json.Marshal(eventPayload{Event: eventType, Timestamp: time.Now().UTC(), Data: dataJSON})
	if err != nil {
		d.logger.Error("failed to marshal webhook envelope", "error", err, "event", eventType)
		return
	}

	for _, hook := range hooks {
		go d.deliverWithRetry(hook, eventType, body)
	}
}

func (d *Dispatcher) deliverWithRetry(hook Webhook, eventType string, body []byte) {
	// Detached from the triggering request's context on purpose: retries can
	// span several seconds via the backoff below, well past when the HTTP
	// request that triggered this event has already returned.
	ctx := context.Background()

	delivery := &Delivery{
		ID:        uuid.New(),
		WebhookID: hook.ID,
		EventType: eventType,
		Payload:   body,
		Status:    DeliveryPending,
	}
	if err := d.repo.CreateDelivery(ctx, delivery); err != nil {
		d.logger.Error("failed to record webhook delivery", "error", err, "webhook_id", hook.ID)
		return
	}

	signature := sign(hook.Secret, body)

	var lastStatus *int
	for attempt := 1; attempt <= maxDeliveryAttempts; attempt++ {
		status, err := d.attempt(hook.URL, body, signature)
		lastStatus = status

		if err == nil && status != nil && *status >= 200 && *status < 300 {
			_ = d.repo.UpdateDelivery(ctx, delivery.ID, DeliverySuccess, attempt, status)
			return
		}

		if attempt < maxDeliveryAttempts {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}

	_ = d.repo.UpdateDelivery(ctx, delivery.ID, DeliveryFailed, maxDeliveryAttempts, lastStatus)
	d.logger.Warn("webhook delivery failed after max attempts", "webhook_id", hook.ID, "url", hook.URL)
}

func (d *Dispatcher) attempt(url string, body []byte, signature string) (*int, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Verifiable proof the payload came from FileForge and wasn't altered
	// in transit - the receiver recomputes this with their own copy of the
	// secret and rejects the request if it doesn't match.
	req.Header.Set("X-FileForge-Signature", signature)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	status := resp.StatusCode
	return &status, nil
}

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
