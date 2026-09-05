package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// WatchQueueDepth polls XLEN for each stream on an interval and publishes it
// as a gauge. Prometheus scrapes a snapshot rather than a stream of events,
// so a periodic poll is the right shape here — there's nothing to count
// incrementally, only a current depth to report.
func WatchQueueDepth(ctx context.Context, logger *slog.Logger, redisURL string, streams []string, interval time.Duration) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error("queue depth watcher: bad redis url", "error", err)
		return
	}
	client := redis.NewClient(opts)
	defer client.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, stream := range streams {
				length, err := client.XLen(ctx, stream).Result()
				if err != nil {
					// A stream that has never been written to doesn't exist
					// yet; that's a depth of zero, not an error worth logging
					// every interval.
					QueueDepth.WithLabelValues(stream).Set(0)
					continue
				}
				QueueDepth.WithLabelValues(stream).Set(float64(length))
			}
		}
	}
}
