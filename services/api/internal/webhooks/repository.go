package webhooks

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("webhook not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, w *Webhook) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO webhooks (id, owner_id, url, event_types, secret, active) VALUES ($1,$2,$3,$4,$5,$6)`,
		w.ID, w.OwnerID, w.URL, w.EventTypes, w.Secret, w.Active,
	)
	return err
}

func (r *Repository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]Webhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, owner_id, url, event_types, secret, active, created_at
		 FROM webhooks WHERE owner_id = $1 ORDER BY created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhooks(rows)
}

// ListActiveForOwnerAndEvent is the dispatch-time lookup: only this owner's
// active webhooks subscribed to this event type, filtered in SQL rather
// than fetched-then-filtered in Go.
func (r *Repository) ListActiveForOwnerAndEvent(ctx context.Context, ownerID uuid.UUID, eventType string) ([]Webhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, owner_id, url, event_types, secret, active, created_at
		 FROM webhooks WHERE owner_id = $1 AND active = true AND $2 = ANY(event_types)`,
		ownerID, eventType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhooks(rows)
}

func (r *Repository) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM webhooks WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id, ownerID uuid.UUID) (*Webhook, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, owner_id, url, event_types, secret, active, created_at
		 FROM webhooks WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	var w Webhook
	err := row.Scan(&w.ID, &w.OwnerID, &w.URL, &w.EventTypes, &w.Secret, &w.Active, &w.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func scanWebhooks(rows pgx.Rows) ([]Webhook, error) {
	var webhooks []Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.OwnerID, &w.URL, &w.EventTypes, &w.Secret, &w.Active, &w.CreatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (r *Repository) CreateDelivery(ctx context.Context, d *Delivery) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event_type, payload, status, attempts)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		d.ID, d.WebhookID, d.EventType, d.Payload, d.Status, d.Attempts,
	)
	return err
}

func (r *Repository) UpdateDelivery(ctx context.Context, id uuid.UUID, status string, attempts int, responseStatus *int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE webhook_deliveries SET status = $1, attempts = $2, response_status = $3, last_attempted_at = now() WHERE id = $4`,
		status, attempts, responseStatus, id,
	)
	return err
}

// ListDeliveries is scoped through webhooks so an owner can only ever see
// delivery logs for a webhook they themselves own.
func (r *Repository) ListDeliveries(ctx context.Context, webhookID, ownerID uuid.UUID, limit int) ([]Delivery, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT d.id, d.webhook_id, d.event_type, d.payload, d.status, d.attempts, d.response_status, d.last_attempted_at, d.created_at
		 FROM webhook_deliveries d
		 JOIN webhooks w ON w.id = d.webhook_id
		 WHERE d.webhook_id = $1 AND w.owner_id = $2
		 ORDER BY d.created_at DESC LIMIT $3`,
		webhookID, ownerID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventType, &d.Payload, &d.Status, &d.Attempts, &d.ResponseStatus, &d.LastAttemptedAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, rows.Err()
}
