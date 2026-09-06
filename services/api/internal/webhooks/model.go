package webhooks

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventJobCompleted   = "job.completed"
	EventJobFailed      = "job.failed"
	EventBatchCompleted = "batch.completed"
)

var AllEventTypes = []string{EventJobCompleted, EventJobFailed, EventBatchCompleted}

type Webhook struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	URL        string
	EventTypes []string
	Secret     string
	Active     bool
	CreatedAt  time.Time
}

const (
	DeliveryPending = "pending"
	DeliverySuccess = "success"
	DeliveryFailed  = "failed"
)

type Delivery struct {
	ID              uuid.UUID
	WebhookID       uuid.UUID
	EventType       string
	Payload         []byte
	Status          string
	Attempts        int
	ResponseStatus  *int
	LastAttemptedAt *time.Time
	CreatedAt       time.Time
}
