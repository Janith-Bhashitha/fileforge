// Package ratelimit implements a Redis-backed fixed-window rate limiter.
//
// It lives in Redis rather than in process memory because Phase 6 runs
// several API replicas behind a load balancer — an in-memory limiter would
// let a caller multiply their allowance by the replica count.
package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
)

type Limiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

func New(redisURL string, limit int, window time.Duration) (*Limiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &Limiter{client: redis.NewClient(opts), limit: limit, window: window}, nil
}

// Allow increments the caller's counter for the current window and reports
// whether they are still under the limit. INCR plus a first-write EXPIRE is
// atomic enough for this: the counter can only ever be over-counted under a
// race, never under-counted, so the limit is never accidentally exceeded.
func (l *Limiter) Allow(ctx context.Context, key string) (allowed bool, remaining int, err error) {
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, time.Now().Unix()/int64(l.window.Seconds()))

	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		// Fail open: a Redis outage should degrade the limiter, not the API.
		return true, l.limit, err
	}
	if count == 1 {
		l.client.Expire(ctx, redisKey, l.window)
	}

	remaining = l.limit - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return int(count) <= l.limit, remaining, nil
}

// Middleware limits per authenticated user, falling back to remote address
// for unauthenticated routes (login/register), which is where credential
// stuffing would show up.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			key = "user:" + claims.UserID.String()
		}

		allowed, remaining, err := l.Allow(r.Context(), key)
		if err == nil {
			w.Header().Set("X-RateLimit-Limit", fmt.Sprint(l.limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprint(remaining))
		}
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprint(int(l.window.Seconds())))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded, please slow down"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return "ip:" + fwd
	}
	return "ip:" + r.RemoteAddr
}
