package jobs

import (
	"time"

	"github.com/google/uuid"
)

// Job status values. These are the only legal values for jobs.status and
// job_items.status — the handlers/worker are what enforce that, since
// Postgres just has a plain TEXT column (a check constraint would work too,
// but keeping the state machine in Go code makes the legal transitions
// easier to see in one place than scattered SQL).
const (
	StatusCreated         = "created"
	StatusQueued          = "queued"
	StatusProcessing      = "processing"
	StatusCompleted       = "completed"
	StatusFailed          = "failed"
	StatusRetryPending    = "retry_pending"
	StatusCancelRequested = "cancel_requested"
	StatusCancelled       = "cancelled"
)

const MaxAttempts = 3

type Job struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Operation string
	Status    string
	Progress  int
	Error     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type JobItem struct {
	ID            uuid.UUID
	JobID         uuid.UUID
	InputFileID   uuid.UUID
	OutputFileID  *uuid.UUID
	Status        string
	Attempts      int
	LastError     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
