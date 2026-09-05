package storage_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
)

// These run against MinIO, which speaks the S3 API, so the exact code path
// that will talk to AWS is what gets exercised — no AWS account, no mocks.
// Skipped unless S3_TEST_ENDPOINT is set, so `go test ./...` stays green on
// a machine with nothing running.
func newTestStore(t *testing.T) *storage.S3Store {
	t.Helper()

	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set; skipping S3 integration test")
	}

	store, err := storage.NewS3Store(context.Background(), storage.S3Config{
		Bucket:          envOr("S3_TEST_BUCKET", "fileforge"),
		Region:          "us-east-1",
		Endpoint:        endpoint,
		ForcePathStyle:  true,
		AccessKeyID:     envOr("S3_TEST_ACCESS_KEY", "minioadmin"),
		SecretAccessKey: envOr("S3_TEST_SECRET_KEY", "minioadmin"),
		WorkDir:         t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewS3Store: %v", err)
	}
	return store
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestS3RoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	content := []byte("hello from fileforge")
	key, err := store.Save(ctx, content, ".txt")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	if !strings.HasSuffix(key, ".txt") {
		t.Errorf("key %q should keep the extension", key)
	}
	// The key must not leak anything about the original file.
	if strings.Contains(key, "hello") {
		t.Errorf("key %q leaks content", key)
	}

	localPath, cleanup, err := store.Fetch(ctx, key)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	got, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read fetched file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("round trip changed content: got %q want %q", got, content)
	}

	cleanup()
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		t.Errorf("cleanup should have removed the scratch file, stat err = %v", err)
	}
}

func TestS3SaveFileConsumesLocalCopy(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Stand in for a processor's output: a file written to scratch space.
	produced := filepath.Join(t.TempDir(), "produced.pdf")
	if err := os.WriteFile(produced, []byte("%PDF-1.4 fake"), 0o644); err != nil {
		t.Fatalf("write produced file: %v", err)
	}

	key, err := store.SaveFile(ctx, produced)
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	if _, err := os.Stat(produced); !os.IsNotExist(err) {
		t.Errorf("SaveFile should have removed the local copy, stat err = %v", err)
	}

	ok, err := store.Exists(ctx, key)
	if err != nil || !ok {
		t.Errorf("uploaded object should exist: ok=%v err=%v", ok, err)
	}
}

func TestS3DeleteRemovesObject(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	key, err := store.Save(ctx, []byte("temporary"), ".txt")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Exists(ctx, key); err == nil {
		t.Error("Exists should error for a deleted object")
	}
}

// The presigned flow is the point of Phase 6: bytes go browser->S3 without
// passing through the API at all.
func TestS3PresignedUploadBypassesAPI(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	url, key, err := store.PresignPut(ctx, ".txt", 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	body := "uploaded straight to object storage"
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("presigned PUT: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("presigned PUT returned %d", resp.StatusCode)
	}

	localPath, cleanup, err := store.Fetch(ctx, key)
	if err != nil {
		t.Fatalf("Fetch after presigned upload: %v", err)
	}
	defer cleanup()

	got, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != body {
		t.Errorf("got %q want %q", got, body)
	}
}

func TestS3PresignedGetIsTimeLimited(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	key, err := store.Save(ctx, []byte("downloadable"), ".txt")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	url, err := store.PresignGet(ctx, key, time.Second)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("presigned GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fresh presigned GET returned %d, want 200", resp.StatusCode)
	}

	// Once the signature's window closes the URL must stop working —
	// otherwise a leaked link is a permanent one.
	time.Sleep(2 * time.Second)
	expired, err := http.Get(url)
	if err != nil {
		t.Fatalf("expired presigned GET: %v", err)
	}
	expired.Body.Close()
	if expired.StatusCode == http.StatusOK {
		t.Error("expired presigned URL still returned 200; expiry is not enforced")
	}
}
