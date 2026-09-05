package batches

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusProcessing        = "processing"
	StatusCompleted         = "completed"
	StatusPartiallyComplete = "partially_completed"
	StatusFailed            = "failed"
)

type Batch struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Operation string
	Options   map[string]string
	Total     int
	Completed int
	Failed    int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
