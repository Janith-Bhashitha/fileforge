package jobs

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("job not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateWithItem inserts a Job and its one JobItem in a single transaction —
// a job row with no corresponding item (or vice versa) should never be
// possible to observe.
func (r *Repository) CreateWithItem(ctx context.Context, job *Job, item *JobItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO jobs (id, owner_id, operation, status, progress) VALUES ($1,$2,$3,$4,$5)`,
		job.ID, job.OwnerID, job.Operation, job.Status, job.Progress,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO job_items (id, job_id, input_file_id, status) VALUES ($1,$2,$3,$4)`,
		item.ID, item.JobID, item.InputFileID, item.Status,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// CreateItemForBatch inserts a single job_item belonging to a batch
// (batch_id set, job_id left null) — the batch row itself is created
// separately by the batches package, which owns that table.
func (r *Repository) CreateItemForBatch(ctx context.Context, item *JobItem) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO job_items (id, batch_id, input_file_id, status) VALUES ($1,$2,$3,$4)`,
		item.ID, item.BatchID, item.InputFileID, item.Status,
	)
	return err
}

func (r *Repository) GetJob(ctx context.Context, id, ownerID uuid.UUID) (*Job, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, owner_id, operation, status, progress, error, created_at, updated_at
		 FROM jobs WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	return scanJob(row)
}

func (r *Repository) ListItemsByJob(ctx context.Context, jobID, ownerID uuid.UUID) ([]JobItem, error) {
	// Join back to jobs to enforce ownership in one query rather than a
	// separate existence check first.
	rows, err := r.pool.Query(ctx,
		`SELECT ji.id, ji.job_id, ji.batch_id, ji.input_file_id, ji.output_file_id, ji.status, ji.attempts, ji.last_error, ji.created_at, ji.updated_at
		 FROM job_items ji
		 JOIN jobs j ON j.id = ji.job_id
		 WHERE ji.job_id = $1 AND j.owner_id = $2
		 ORDER BY ji.created_at`,
		jobID, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

// ListItemsByBatch mirrors ListItemsByJob for the batch case.
func (r *Repository) ListItemsByBatch(ctx context.Context, batchID, ownerID uuid.UUID) ([]JobItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ji.id, ji.job_id, ji.batch_id, ji.input_file_id, ji.output_file_id, ji.status, ji.attempts, ji.last_error, ji.created_at, ji.updated_at
		 FROM job_items ji
		 JOIN batches b ON b.id = ji.batch_id
		 WHERE ji.batch_id = $1 AND b.owner_id = $2
		 ORDER BY ji.created_at`,
		batchID, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

// GetItemByID has no owner check — this is used by workers, which act on
// behalf of the system, not a specific authenticated request.
func (r *Repository) GetItemByID(ctx context.Context, id uuid.UUID) (*JobItem, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, job_id, batch_id, input_file_id, output_file_id, status, attempts, last_error, created_at, updated_at
		 FROM job_items WHERE id = $1`,
		id,
	)
	var item JobItem
	err := row.Scan(&item.ID, &item.JobID, &item.BatchID, &item.InputFileID, &item.OutputFileID, &item.Status, &item.Attempts, &item.LastError, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpdateJobStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $1, error = $2, updated_at = now() WHERE id = $3`,
		status, errMsg, id,
	)
	return err
}

func (r *Repository) UpdateJobProgress(ctx context.Context, id uuid.UUID, progress int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET progress = $1, updated_at = now() WHERE id = $2`,
		progress, id,
	)
	return err
}

// UpdateItem is the one place worker code writes item outcomes — a single
// call covers "processing started", "completed with an output file",
// "failed with an attempt increment", etc., depending on which pointers are
// non-nil. Works identically whether the item belongs to a Job or a Batch.
func (r *Repository) UpdateItem(ctx context.Context, id uuid.UUID, status string, outputFileID *uuid.UUID, lastError *string, incrementAttempts bool) error {
	if incrementAttempts {
		_, err := r.pool.Exec(ctx,
			`UPDATE job_items SET status = $1, output_file_id = $2, last_error = $3, attempts = attempts + 1, updated_at = now() WHERE id = $4`,
			status, outputFileID, lastError, id,
		)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE job_items SET status = $1, output_file_id = $2, last_error = $3, updated_at = now() WHERE id = $4`,
		status, outputFileID, lastError, id,
	)
	return err
}

// CancelIfQueued cancels a job only while it's still safe to do so — once
// a worker has picked it up (processing or later), cancellation is no
// longer just a database flip. Returns false if the job wasn't in a
// cancellable state.
func (r *Repository) CancelIfQueued(ctx context.Context, id, ownerID uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $1, updated_at = now()
		 WHERE id = $2 AND owner_id = $3 AND status IN ($4, $5)`,
		StatusCancelled, id, ownerID, StatusCreated, StatusQueued,
	)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE job_items SET status = $1, updated_at = now() WHERE job_id = $2 AND status = $3`,
		StatusCancelled, id, StatusQueued,
	)
	return true, err
}

// ResetFailedItemsForRetry flips a job's failed item(s) back to queued so
// the caller can re-enqueue them. Returns the items that were reset, so the
// caller knows exactly what to push back onto the stream.
func (r *Repository) ResetFailedItemsForRetry(ctx context.Context, jobID, ownerID uuid.UUID) ([]JobItem, error) {
	rows, err := r.pool.Query(ctx,
		`UPDATE job_items SET status = $1, updated_at = now()
		 WHERE job_id = $2 AND status = $3
		   AND job_id IN (SELECT id FROM jobs WHERE id = $2 AND owner_id = $4)
		 RETURNING id, job_id, batch_id, input_file_id, output_file_id, status, attempts, last_error, created_at, updated_at`,
		StatusQueued, jobID, StatusFailed, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanItems(rows)
	if err != nil {
		return nil, err
	}

	if len(items) > 0 {
		if _, err := r.pool.Exec(ctx, `UPDATE jobs SET status = $1, error = NULL, updated_at = now() WHERE id = $2`, StatusQueued, jobID); err != nil {
			return nil, err
		}
	}

	return items, nil
}

// ResetFailedItemsForBatch mirrors ResetFailedItemsForRetry for the batch
// case — flips failed items back to queued so the caller can re-enqueue
// them, scoped to a batch the owner actually owns.
func (r *Repository) ResetFailedItemsForBatch(ctx context.Context, batchID, ownerID uuid.UUID) ([]JobItem, error) {
	rows, err := r.pool.Query(ctx,
		`UPDATE job_items SET status = $1, updated_at = now()
		 WHERE batch_id = $2 AND status = $3
		   AND batch_id IN (SELECT id FROM batches WHERE id = $2 AND owner_id = $4)
		 RETURNING id, job_id, batch_id, input_file_id, output_file_id, status, attempts, last_error, created_at, updated_at`,
		StatusQueued, batchID, StatusFailed, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

func scanItems(rows pgx.Rows) ([]JobItem, error) {
	var items []JobItem
	for rows.Next() {
		var item JobItem
		if err := rows.Scan(&item.ID, &item.JobID, &item.BatchID, &item.InputFileID, &item.OutputFileID, &item.Status, &item.Attempts, &item.LastError, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanJob(row pgx.Row) (*Job, error) {
	var j Job
	err := row.Scan(&j.ID, &j.OwnerID, &j.Operation, &j.Status, &j.Progress, &j.Error, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &j, nil
}
