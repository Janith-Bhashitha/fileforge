// Package audit records who did what, for the security-relevant actions
// (auth, file lifecycle, job submission). It is deliberately best-effort:
// an audit write failing must never fail the user's actual request, so
// Record logs the problem and moves on rather than returning an error into
// the request path.
package audit

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ActionUserRegistered = "user.registered"
	ActionUserLoggedIn   = "user.logged_in"
	ActionLoginFailed    = "user.login_failed"
	ActionFileUploaded   = "file.uploaded"
	ActionFileDeleted    = "file.deleted"
	ActionFileConverted  = "file.converted"
	ActionJobCreated     = "job.created"
	ActionJobCancelled   = "job.cancelled"
	ActionBatchCreated   = "batch.created"
	ActionQuotaExceeded  = "quota.exceeded"
)

type Recorder struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewRecorder(pool *pgxpool.Pool, logger *slog.Logger) *Recorder {
	return &Recorder{pool: pool, logger: logger}
}

type Event struct {
	UserID       *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	Metadata     map[string]any
}

func (r *Recorder) Record(ctx context.Context, e Event) {
	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_events (id, user_id, action, resource_type, resource_id, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		uuid.New(), e.UserID, e.Action, nullIfEmpty(e.ResourceType), e.ResourceID, e.Metadata,
	)
	if err != nil {
		r.logger.Error("audit write failed", "action", e.Action, "error", err)
	}
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
