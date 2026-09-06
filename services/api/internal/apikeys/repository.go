package apikeys

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("api key not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, k *APIKey) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO api_keys (id, owner_id, name, key_prefix, key_hash) VALUES ($1,$2,$3,$4,$5)`,
		k.ID, k.OwnerID, k.Name, k.KeyPrefix, k.KeyHash,
	)
	return err
}

func (r *Repository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, owner_id, name, key_prefix, key_hash, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE owner_id = $1 ORDER BY created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.OwnerID, &k.Name, &k.KeyPrefix, &k.KeyHash, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// CandidatesByPrefix is the first step of verifying an incoming key: narrow
// to the handful of active keys sharing this prefix (in practice almost
// always zero or one) before ever calling bcrypt, which is deliberately
// slow and shouldn't run once per stored key on every request.
func (r *Repository) CandidatesByPrefix(ctx context.Context, prefix string) ([]APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, owner_id, name, key_prefix, key_hash, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE key_prefix = $1 AND revoked_at IS NULL`,
		prefix,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.OwnerID, &k.Name, &k.KeyPrefix, &k.KeyHash, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *Repository) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
	return err
}

// Revoke is scoped to the owner so one user can never revoke another's key
// by guessing an ID.
func (r *Repository) Revoke(ctx context.Context, id, ownerID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND owner_id = $2 AND revoked_at IS NULL`,
		id, ownerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
