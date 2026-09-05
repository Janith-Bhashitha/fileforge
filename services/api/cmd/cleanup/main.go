// Command cleanup deletes files past their retention window and expires
// jobs that were left stranded (queued or processing with nothing working
// on them, e.g. after a worker crash — the gap the consumer's missing
// XPENDING/XCLAIM sweep leaves behind).
//
// It runs to completion and exits, so it can be driven by cron now and by
// an ECS scheduled task in Phase 6 without changing anything but the
// scheduler.
package main

import (
	"context"
	"time"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/config"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/db"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/logging"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
)

func main() {
	logger := logging.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		return
	}
	defer pool.Close()

	store, err := storage.NewLocalStore(cfg.StorageDir)
	if err != nil {
		logger.Error("failed to init storage", "error", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -cfg.RetentionDays)
	logger.Info("cleanup starting", "retention_days", cfg.RetentionDays, "cutoff", cutoff)

	// Expire stranded work first: a job stuck in "processing" since before
	// the cutoff has no worker behind it any more, and leaving it in a
	// non-terminal state means its quota slot is never released either.
	stranded, err := pool.Exec(ctx,
		`UPDATE jobs SET status = 'expired', error = 'expired by cleanup: no worker completed this job',
		     updated_at = now()
		 WHERE status IN ('created','queued','processing','retry_pending') AND created_at < $1`,
		cutoff,
	)
	if err != nil {
		logger.Error("failed to expire stranded jobs", "error", err)
	} else {
		logger.Info("expired stranded jobs", "count", stranded.RowsAffected())
	}

	strandedItems, err := pool.Exec(ctx,
		`UPDATE job_items SET status = 'failed', last_error = 'expired by cleanup', updated_at = now()
		 WHERE status IN ('created','queued','processing','retry_pending') AND created_at < $1`,
		cutoff,
	)
	if err != nil {
		logger.Error("failed to expire stranded job items", "error", err)
	} else {
		logger.Info("expired stranded job items", "count", strandedItems.RowsAffected())
	}

	// Then delete expired files. Storage objects are removed first: an
	// orphaned row with no object is recoverable noise, while an orphaned
	// object with no row is invisible and leaks disk forever.
	rows, err := pool.Query(ctx,
		`SELECT id, storage_key FROM files WHERE created_at < $1`, cutoff)
	if err != nil {
		logger.Error("failed to list expired files", "error", err)
		return
	}

	type expired struct {
		id  string
		key string
	}
	var toDelete []expired
	for rows.Next() {
		var e expired
		if err := rows.Scan(&e.id, &e.key); err != nil {
			logger.Error("failed to scan expired file", "error", err)
			continue
		}
		toDelete = append(toDelete, e)
	}
	rows.Close()

	deleted := 0
	for _, e := range toDelete {
		if err := store.Delete(ctx, e.key); err != nil {
			logger.Warn("failed to delete storage object", "storage_key", e.key, "error", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM files WHERE id = $1`, e.id); err != nil {
			logger.Error("failed to delete file row", "file_id", e.id, "error", err)
			continue
		}
		deleted++
	}

	logger.Info("cleanup finished", "files_deleted", deleted, "files_considered", len(toDelete))
}
