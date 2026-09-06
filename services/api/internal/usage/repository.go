// Package usage reports per-account activity from the files table.
//
// files is the source rather than jobs: the synchronous /convert endpoint
// never creates a job row, so job_items would miss most real activity. Every
// conversion, sync or queued, writes an output file with operation set.
package usage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type DailyPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type OperationCount struct {
	Operation string `json:"operation"`
	Count     int    `json:"count"`
}

type Summary struct {
	Conversions  int              `json:"conversions"`
	Uploads      int              `json:"uploads"`
	BytesIn      int64            `json:"bytes_in"`
	BytesStored  int64            `json:"bytes_stored"`
	Daily        []DailyPoint     `json:"daily"`
	TopOperation []OperationCount `json:"top_operations"`
}

func (r *Repository) Summarize(ctx context.Context, ownerID uuid.UUID, since time.Time) (*Summary, error) {
	s := &Summary{Daily: []DailyPoint{}, TopOperation: []OperationCount{}}

	err := r.pool.QueryRow(ctx,
		`SELECT
		   count(*) FILTER (WHERE operation IS NOT NULL),
		   count(*) FILTER (WHERE operation IS NULL),
		   coalesce(sum(size), 0)
		 FROM files WHERE owner_id = $1 AND created_at >= $2`,
		ownerID, since,
	).Scan(&s.Conversions, &s.Uploads, &s.BytesIn)
	if err != nil {
		return nil, err
	}

	// Storage is deliberately all-time, not windowed: it answers "how much am
	// I holding right now", which a date filter would misreport.
	if err := r.pool.QueryRow(ctx,
		`SELECT coalesce(sum(size), 0) FROM files WHERE owner_id = $1`, ownerID,
	).Scan(&s.BytesStored); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day, count(*)
		 FROM files
		 WHERE owner_id = $1 AND operation IS NOT NULL AND created_at >= $2
		 GROUP BY day ORDER BY day`,
		ownerID, since,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Count); err != nil {
			rows.Close()
			return nil, err
		}
		s.Daily = append(s.Daily, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	opRows, err := r.pool.Query(ctx,
		`SELECT operation, count(*) AS n
		 FROM files
		 WHERE owner_id = $1 AND operation IS NOT NULL AND created_at >= $2
		 GROUP BY operation ORDER BY n DESC LIMIT 6`,
		ownerID, since,
	)
	if err != nil {
		return nil, err
	}
	defer opRows.Close()
	for opRows.Next() {
		var o OperationCount
		if err := opRows.Scan(&o.Operation, &o.Count); err != nil {
			return nil, err
		}
		s.TopOperation = append(s.TopOperation, o)
	}
	return s, opRows.Err()
}
