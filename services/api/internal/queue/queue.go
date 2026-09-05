package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Message is what gets written to a stream and read back by a worker. It
// carries everything a worker needs to run one JobItem without a database
// round trip first (the worker still writes results back to Postgres, but
// doesn't need to query it just to know what to do).
type Message struct {
	JobItemID   string            `json:"job_item_id"`
	Operation   string            `json:"operation"`
	Version     string            `json:"version"`
	InputFileID string            `json:"input_file_id"`
	Options     map[string]string `json:"options"`
}

// StreamForOperation maps an operation name to the stream its worker type
// consumes. This is an explicit table, not a prefix guess — "pdf-to-image"
// and "image-to-pdf" both belong to the image worker despite one containing
// "pdf" in its name, so string-prefix matching would get this wrong.
var StreamForOperation = map[string]string{
	"image-to-pdf":  "stream:image",
	"pdf-to-image":  "stream:image",
	"image-convert": "stream:image",
	"image-resize":  "stream:image",
	"docx-to-pdf":   "stream:office",
	"pptx-to-pdf":   "stream:office",
	"xlsx-to-pdf":   "stream:office",
	"txt-to-pdf":    "stream:office",
	"pdf-merge":     "stream:pdf",
	"pdf-split":     "stream:pdf",
	"pdf-compress":  "stream:pdf",
}

type Producer struct {
	client *redis.Client
}

func NewProducer(redisURL string) (*Producer, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &Producer{client: redis.NewClient(opts)}, nil
}

// Enqueue writes msg onto the stream for its operation. Returns an error if
// the operation isn't in StreamForOperation at all (caller should validate
// against the operation registry before ever reaching this point, but this
// is a second line of defense against a job silently going nowhere).
func (p *Producer) Enqueue(ctx context.Context, msg Message) error {
	stream, ok := StreamForOperation[msg.Operation]
	if !ok {
		return fmt.Errorf("no stream configured for operation: %s", msg.Operation)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{"payload": string(data)},
	}).Err()
}
