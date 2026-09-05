package batches

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("batch not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, b *Batch) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO batches (id, owner_id, operation, total, status) VALUES ($1,$2,$3,$4,$5)`,
		b.ID, b.OwnerID, b.Operation, b.Total, b.Status,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id, ownerID uuid.UUID) (*Batch, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, owner_id, operation, total, completed, failed, status, created_at, updated_at
		 FROM batches WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	return scanBatch(row)
}

// IncrementCompleted and IncrementFailed each do the counter bump and the
// status recompute in one atomic UPDATE, so concurrent workers finishing
// items from the same batch at the same time can never race each other
// into an inconsistent count (Postgres's row-level lock on the UPDATE
// serializes them).
func (r *Repository) IncrementCompleted(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE batches
		SET completed = completed + 1,
		    updated_at = now(),
		    status = CASE
		        WHEN completed + 1 + failed >= total THEN
		            CASE WHEN failed = 0 THEN 'completed' ELSE 'partially_completed' END
		        ELSE 'processing'
		    END
		WHERE id = $1`,
		id,
	)
	return err
}

func (r *Repository) IncrementFailed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE batches
		SET failed = failed + 1,
		    updated_at = now(),
		    status = CASE
		        WHEN completed + failed + 1 >= total THEN
		            CASE WHEN completed = 0 THEN 'failed' ELSE 'partially_completed' END
		        ELSE 'processing'
		    END
		WHERE id = $1`,
		id,
	)
	return err
}

// ResetForRetry undoes count worth of failed increments and puts the batch
// back into "processing" so its status reflects that work is outstanding
// again, ahead of the caller re-enqueuing those items.
func (r *Repository) ResetForRetry(ctx context.Context, id uuid.UUID, count int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE batches SET failed = failed - $1, status = $2, updated_at = now() WHERE id = $3`,
		count, StatusProcessing, id,
	)
	return err
}

func scanBatch(row pgx.Row) (*Batch, error) {
	var b Batch
	err := row.Scan(&b.ID, &b.OwnerID, &b.Operation, &b.Total, &b.Completed, &b.Failed, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}
