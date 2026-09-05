// Package workerconsumer is the shared engine behind every worker binary
// (worker-pdf, worker-image, worker-office). Each worker's main.go is just
// a few lines of wiring that builds a Runner pointed at its own stream and
// calls Run — all the actual consume/process/update logic lives here once,
// not duplicated three times.
package workerconsumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/convert"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/files"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/jobs"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/queue"
	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
)

type Runner struct {
	Logger    *slog.Logger
	Redis     *redis.Client
	Producer  *queue.Producer
	Stream    string
	Group     string
	Consumer  string
	Registry  *convert.Registry
	JobsRepo  *jobs.Repository
	FilesRepo *files.Repository
	Store     storage.Store
}

// Run blocks forever, consuming messages from r.Stream until ctx is
// cancelled. Known gap (disclosed, not hidden): if this process crashes
// mid-message, that message stays "pending" in Redis with no automatic
// reclaim — Phase 3 doesn't implement the XPENDING/XCLAIM recovery sweep
// the full plan calls for. A stuck item would need manual intervention.
func (r *Runner) Run(ctx context.Context) error {
	err := r.Redis.XGroupCreateMkStream(ctx, r.Stream, r.Group, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group: %w", err)
	}

	r.Logger.Info("worker started", "stream", r.Stream, "group", r.Group, "consumer", r.Consumer)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		streams, err := r.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    r.Group,
			Consumer: r.Consumer,
			Streams:  []string{r.Stream, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			r.Logger.Error("xreadgroup failed", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				r.handleMessage(ctx, msg)
			}
		}
	}
}

func (r *Runner) handleMessage(ctx context.Context, msg redis.XMessage) {
	defer func() {
		if err := r.Redis.XAck(ctx, r.Stream, r.Group, msg.ID).Err(); err != nil {
			r.Logger.Error("xack failed", "error", err, "msg_id", msg.ID)
		}
	}()

	payload, ok := msg.Values["payload"].(string)
	if !ok {
		r.Logger.Error("message missing payload field", "msg_id", msg.ID)
		return
	}

	var qmsg queue.Message
	if err := json.Unmarshal([]byte(payload), &qmsg); err != nil {
		r.Logger.Error("failed to unmarshal message", "error", err)
		return
	}

	itemID, err := uuid.Parse(qmsg.JobItemID)
	if err != nil {
		r.Logger.Error("invalid job_item_id in message", "error", err)
		return
	}

	item, err := r.JobsRepo.GetItemByID(ctx, itemID)
	if err != nil {
		r.Logger.Error("failed to fetch job item", "error", err, "job_item_id", itemID)
		return
	}

	// Idempotency guard: only ever process an item that's still queued.
	// Protects against a message somehow being delivered more than once.
	if item.Status != jobs.StatusQueued {
		r.Logger.Info("skipping item not in queued state", "job_item_id", itemID, "status", item.Status)
		return
	}

	logger := r.Logger.With("job_id", item.JobID, "job_item_id", item.ID, "operation", qmsg.Operation, "worker", r.Consumer)

	_ = r.JobsRepo.UpdateItem(ctx, item.ID, jobs.StatusProcessing, nil, nil, false)
	nilErr := (*string)(nil)
	_ = r.JobsRepo.UpdateJobStatus(ctx, item.JobID, jobs.StatusProcessing, nilErr)

	inputFileID, err := uuid.Parse(qmsg.InputFileID)
	if err != nil {
		r.failItem(ctx, item, "invalid input file id", logger)
		return
	}

	inputFile, err := r.FilesRepo.GetByIDAny(ctx, inputFileID)
	if err != nil {
		r.failItem(ctx, item, "input file not found: "+err.Error(), logger)
		return
	}

	processor, err := r.Registry.Resolve(qmsg.Operation, qmsg.Version)
	if err != nil {
		r.failItem(ctx, item, err.Error(), logger)
		return
	}

	result, err := processor.Process(ctx, convert.ConversionRequest{
		InputPath: r.Store.LocalPath(inputFile.StorageKey),
		Options:   qmsg.Options,
	})
	if err != nil {
		r.retryOrFail(ctx, item, qmsg, "conversion failed: "+err.Error(), logger)
		return
	}

	outputKey := filepath.Base(result.OutputPath)
	var size int64
	if info, statErr := os.Stat(result.OutputPath); statErr == nil {
		size = info.Size()
	}

	displayName := convert.DisplayFilename(inputFile.Filename, qmsg.Operation, filepath.Ext(result.OutputPath))
	operationCopy := qmsg.Operation

	outputFile := &files.File{
		ID:                uuid.New(),
		OwnerID:           inputFile.OwnerID,
		Filename:          displayName,
		MimeType:          result.MimeType,
		Size:              size,
		StorageKey:        outputKey,
		DerivedFromFileID: &inputFile.ID,
		Operation:         &operationCopy,
	}

	if err := r.FilesRepo.Create(ctx, outputFile); err != nil {
		r.failItem(ctx, item, "failed to save output file: "+err.Error(), logger)
		return
	}

	_ = r.JobsRepo.UpdateItem(ctx, item.ID, jobs.StatusCompleted, &outputFile.ID, nil, false)
	_ = r.JobsRepo.UpdateJobStatus(ctx, item.JobID, jobs.StatusCompleted, nilErr)
	_ = r.JobsRepo.UpdateJobProgress(ctx, item.JobID, 100)

	logger.Info("job item completed", "output_file_id", outputFile.ID)
}

// retryOrFail implements the simplified retry policy disclosed up front:
// up to jobs.MaxAttempts, with a short in-process backoff sleep between
// attempts (this blocks the worker from picking up its next message during
// the sleep - acceptable at this project's scale, not how Phase 3's full
// design would do it at higher volume).
func (r *Runner) retryOrFail(ctx context.Context, item *jobs.JobItem, qmsg queue.Message, errMsg string, logger *slog.Logger) {
	attempts := item.Attempts + 1
	if attempts >= jobs.MaxAttempts {
		r.failItem(ctx, item, errMsg, logger)
		return
	}

	backoff := time.Duration(attempts) * 2 * time.Second
	logger.Warn("job item failed, retrying", "attempt", attempts, "backoff", backoff, "error", errMsg)

	_ = r.JobsRepo.UpdateItem(ctx, item.ID, jobs.StatusRetryPending, nil, &errMsg, true)
	time.Sleep(backoff)
	_ = r.JobsRepo.UpdateItem(ctx, item.ID, jobs.StatusQueued, nil, &errMsg, false)

	if err := r.Producer.Enqueue(ctx, qmsg); err != nil {
		logger.Error("failed to re-enqueue after retry backoff", "error", err)
		r.failItem(ctx, item, "failed to re-enqueue: "+err.Error(), logger)
	}
}

func (r *Runner) failItem(ctx context.Context, item *jobs.JobItem, errMsg string, logger *slog.Logger) {
	logger.Error("job item failed permanently", "error", errMsg)
	_ = r.JobsRepo.UpdateItem(ctx, item.ID, jobs.StatusFailed, nil, &errMsg, true)
	_ = r.JobsRepo.UpdateJobStatus(ctx, item.JobID, jobs.StatusFailed, &errMsg)
}
