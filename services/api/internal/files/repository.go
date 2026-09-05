package files

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("file not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, f *File) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO files (id, owner_id, filename, mime_type, size, checksum, storage_key, derived_from_file_id, operation)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		f.ID, f.OwnerID, f.Filename, f.MimeType, f.Size, f.Checksum, f.StorageKey, f.DerivedFromFileID, f.Operation,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id, ownerID uuid.UUID) (*File, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, owner_id, filename, mime_type, size, checksum, storage_key, derived_from_file_id, operation, created_at
		 FROM files WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	return scanFile(row)
}

// GetByIDAny looks up a file with no owner check — for worker code, which
// acts on behalf of the system rather than a specific authenticated
// request. Every HTTP-facing path must keep using GetByID instead.
func (r *Repository) GetByIDAny(ctx context.Context, id uuid.UUID) (*File, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, owner_id, filename, mime_type, size, checksum, storage_key, derived_from_file_id, operation, created_at
		 FROM files WHERE id = $1`,
		id,
	)
	return scanFile(row)
}

// ListByOwner returns every file the user owns (uploads and conversion
// outputs alike), newest first, each annotated with how many other files
// were derived from it.
func (r *Repository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]ListItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT f.id, f.owner_id, f.filename, f.mime_type, f.size, f.checksum, f.storage_key,
		        f.derived_from_file_id, f.operation, f.created_at,
		        (SELECT COUNT(*) FROM files c WHERE c.derived_from_file_id = f.id) AS derived_count
		 FROM files f
		 WHERE f.owner_id = $1
		 ORDER BY f.created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ListItem
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(
			&item.ID, &item.OwnerID, &item.Filename, &item.MimeType, &item.Size, &item.Checksum,
			&item.StorageKey, &item.DerivedFromFileID, &item.Operation, &item.CreatedAt, &item.DerivedCount,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM files WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanFile(row pgx.Row) (*File, error) {
	var f File
	err := row.Scan(&f.ID, &f.OwnerID, &f.Filename, &f.MimeType, &f.Size, &f.Checksum, &f.StorageKey, &f.DerivedFromFileID, &f.Operation, &f.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}
