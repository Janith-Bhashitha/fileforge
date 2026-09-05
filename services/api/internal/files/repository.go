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
		`INSERT INTO files (id, owner_id, filename, mime_type, size, checksum, storage_key, derived_from_file_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		f.ID, f.OwnerID, f.Filename, f.MimeType, f.Size, f.Checksum, f.StorageKey, f.DerivedFromFileID,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id, ownerID uuid.UUID) (*File, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, owner_id, filename, mime_type, size, checksum, storage_key, derived_from_file_id, created_at
		 FROM files WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	return scanFile(row)
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
	err := row.Scan(&f.ID, &f.OwnerID, &f.Filename, &f.MimeType, &f.Size, &f.Checksum, &f.StorageKey, &f.DerivedFromFileID, &f.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}
