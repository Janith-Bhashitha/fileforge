// Package quota caps how much work one user can have in flight at once.
//
// The count lives in Redis, incremented when work is accepted and
// decremented when it reaches a terminal state, so checking it is a single
// O(1) read rather than a COUNT(*) over the jobs table on every submission.
package quota

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Tracker struct {
	client        *redis.Client
	maxConcurrent int
}

func New(redisURL string, maxConcurrent int) (*Tracker, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &Tracker{client: redis.NewClient(opts), maxConcurrent: maxConcurrent}, nil
}

func (t *Tracker) Max() int { return t.maxConcurrent }

func key(userID uuid.UUID) string { return "quota:inflight:" + userID.String() }

// Reserve claims n slots if the user has them free, and reports how many
// they were actually allowed. It increments first and rolls back on
// overshoot, so two concurrent submissions can't both read "under the
// limit" and both proceed.
func (t *Tracker) Reserve(ctx context.Context, userID uuid.UUID, n int) (bool, error) {
	count, err := t.client.IncrBy(ctx, key(userID), int64(n)).Result()
	if err != nil {
		// Fail open — a Redis outage shouldn't block legitimate work.
		return true, err
	}

	if int(count) > t.maxConcurrent {
		t.client.DecrBy(ctx, key(userID), int64(n))
		return false, nil
	}
	return true, nil
}

// Release gives slots back when work reaches a terminal state. The counter
// is floored at zero so a double-release (a retried worker, a replayed
// message) can't drive it negative and hand out free capacity forever.
func (t *Tracker) Release(ctx context.Context, userID uuid.UUID, n int) error {
	count, err := t.client.DecrBy(ctx, key(userID), int64(n)).Result()
	if err != nil {
		return err
	}
	if count < 0 {
		t.client.Set(ctx, key(userID), 0, 0)
	}
	return nil
}

func (t *Tracker) InFlight(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := t.client.Get(ctx, key(userID)).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}
