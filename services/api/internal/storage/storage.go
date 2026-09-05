package storage

import (
	"context"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Store persists file bytes and resolves a key back to a local, readable
// path. This local implementation is what Phase 6 swaps for S3 — callers
// only ever deal with keys and local paths, never storage-backend specifics.
type Store interface {
	Save(ctx context.Context, data []byte, ext string) (key string, err error)
	LocalPath(key string) string
	Delete(ctx context.Context, key string) error
}

type LocalStore struct {
	baseDir string
}

func NewLocalStore(baseDir string) (*LocalStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &LocalStore{baseDir: baseDir}, nil
}

func (s *LocalStore) Save(_ context.Context, data []byte, ext string) (string, error) {
	key := uuid.New().String() + ext
	path := filepath.Join(s.baseDir, key)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return key, nil
}

func (s *LocalStore) LocalPath(key string) string {
	return filepath.Join(s.baseDir, key)
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	return os.Remove(s.LocalPath(key))
}

// BaseDir exposes the directory processors should write outputs into so
// their generated filenames can be treated directly as storage keys.
func (s *LocalStore) BaseDir() string {
	return s.baseDir
}
