package users

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")
var ErrEmailTaken = errors.New("email already registered")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, u *User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1, $2, $3, $4)`,
		u.ID, u.Email, u.PasswordHash, u.DisplayName,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return err
	}
	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, avatar_key, reset_token, reset_token_expires_at, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	)
	return scanUser(row)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, avatar_key, reset_token, reset_token_expires_at, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	)
	return scanUser(row)
}

func (r *Repository) GetByResetToken(ctx context.Context, token string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, avatar_key, reset_token, reset_token_expires_at, created_at, updated_at
		 FROM users WHERE reset_token = $1`,
		token,
	)
	return scanUser(row)
}

// UpdateProfile is the endpoint that didn't exist before - display_name was
// write-once at registration. Both fields are updated together since the
// settings form submits them together; pass the existing value for one to
// leave it unchanged.
func (r *Repository) UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, avatarKey *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET display_name = $1, avatar_key = $2, updated_at = now() WHERE id = $3`,
		displayName, avatarKey, id,
	)
	return err
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`,
		passwordHash, id,
	)
	return err
}

// SetResetToken and ClearResetToken bookend the password-reset flow: a
// token is issued with an expiry, and cleared the moment it's used (or
// replaced by a fresh request) so it can never be reused.
func (r *Repository) SetResetToken(ctx context.Context, id uuid.UUID, token string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET reset_token = $1, reset_token_expires_at = $2, updated_at = now() WHERE id = $3`,
		token, expiresAt, id,
	)
	return err
}

func (r *Repository) ClearResetToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET reset_token = NULL, reset_token_expires_at = NULL, updated_at = now() WHERE id = $1`,
		id,
	)
	return err
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarKey, &u.ResetToken, &u.ResetTokenExpiresAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
