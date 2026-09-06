package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/storage"
)

// Unlike the S3 tests these need nothing running, so they are what guards
// the owner-prefix logic on an ordinary `go test ./...`.

func newLocalStore(t *testing.T) *storage.LocalStore {
	t.Helper()
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	return store
}

func TestLocalSaveNamespacesByOwner(t *testing.T) {
	store := newLocalStore(t)
	ctx := context.Background()
	owner := uuid.New()

	content := []byte("hello from fileforge")
	key, err := store.Save(ctx, owner, content, ".txt")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if want := "users/" + owner.String() + "/"; !strings.HasPrefix(key, want) {
		t.Errorf("key %q should start with %q", key, want)
	}
	if !strings.HasSuffix(key, ".txt") {
		t.Errorf("key %q should keep the extension", key)
	}

	localPath, cleanup, err := store.Fetch(ctx, key)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	defer cleanup()

	got, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("round trip changed content: got %q want %q", got, content)
	}
}

func TestLocalSaveSeparatesOwners(t *testing.T) {
	store := newLocalStore(t)
	ctx := context.Background()

	keyA, err := store.Save(ctx, uuid.New(), []byte("a"), ".txt")
	if err != nil {
		t.Fatalf("Save A: %v", err)
	}
	keyB, err := store.Save(ctx, uuid.New(), []byte("b"), ".txt")
	if err != nil {
		t.Fatalf("Save B: %v", err)
	}

	dirA, dirB := filepath.Dir(keyA), filepath.Dir(keyB)
	if dirA == dirB {
		t.Errorf("different owners shared a folder: %q", dirA)
	}
}

func TestLocalSaveFileMovesProducedFileUnderOwner(t *testing.T) {
	store := newLocalStore(t)
	ctx := context.Background()
	owner := uuid.New()

	// Stands in for a processor's output, written into the store's work dir.
	produced := filepath.Join(store.WorkDir(), "produced.pdf")
	if err := os.WriteFile(produced, []byte("%PDF-1.4 fake"), 0o644); err != nil {
		t.Fatalf("write produced file: %v", err)
	}

	key, err := store.SaveFile(ctx, owner, produced)
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	if want := "users/" + owner.String() + "/"; !strings.HasPrefix(key, want) {
		t.Errorf("key %q should start with %q", key, want)
	}
	if _, err := os.Stat(produced); !os.IsNotExist(err) {
		t.Errorf("SaveFile should have moved the produced file, stat err = %v", err)
	}

	localPath, cleanup, err := store.Fetch(ctx, key)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	defer cleanup()
	if _, err := os.Stat(localPath); err != nil {
		t.Errorf("stored file should exist: %v", err)
	}
}

// Keys are stored in full, so objects written before the owner prefix
// existed must keep working - that is what makes it safe without a backfill.
func TestLocalFetchStillReadsLegacyFlatKeys(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}

	legacyKey := uuid.New().String() + ".txt"
	if err := os.WriteFile(filepath.Join(dir, legacyKey), []byte("old object"), 0o644); err != nil {
		t.Fatalf("seed legacy object: %v", err)
	}

	localPath, cleanup, err := store.Fetch(context.Background(), legacyKey)
	if err != nil {
		t.Fatalf("Fetch legacy key: %v", err)
	}
	defer cleanup()

	got, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read legacy object: %v", err)
	}
	if string(got) != "old object" {
		t.Errorf("got %q want %q", got, "old object")
	}
}

func TestLocalRejectsTraversalKeys(t *testing.T) {
	store := newLocalStore(t)
	ctx := context.Background()

	for _, key := range []string{"../escaped.txt", "users/../../escaped.txt"} {
		if _, _, err := store.Fetch(ctx, key); err == nil {
			t.Errorf("Fetch(%q) should have been rejected", key)
		}
		if err := store.Delete(ctx, key); err == nil {
			t.Errorf("Delete(%q) should have been rejected", key)
		}
	}
}

func TestLocalDeleteRemovesObject(t *testing.T) {
	store := newLocalStore(t)
	ctx := context.Background()

	key, err := store.Save(ctx, uuid.New(), []byte("temporary"), ".txt")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(store.LocalPath(key)); !os.IsNotExist(err) {
		t.Errorf("object should be gone, stat err = %v", err)
	}
}
