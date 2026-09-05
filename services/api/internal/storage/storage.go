package storage

import (
	"context"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Store persists file bytes and hands back an opaque key.
//
// Conversion work always happens on real files on disk — LibreOffice,
// pdfcpu and pdftoppm all take paths, not readers — so the interface is
// built around that fact rather than pretending everything is a stream:
//
//	Fetch  gets a key onto local disk to be worked on, and hands back the
//	       cleanup that releases it.
//	Save / SaveFile put bytes or a produced file back, returning its key.
//
// LocalStore satisfies this with no copying at all (Fetch is a path lookup,
// cleanup a no-op); S3Store downloads and uploads around the same calls.
// Nothing above this interface knows which one it's talking to.
type Store interface {
	Save(ctx context.Context, data []byte, ext string) (key string, err error)
	SaveFile(ctx context.Context, localPath string) (key string, err error)
	Fetch(ctx context.Context, key string) (localPath string, cleanup func(), err error)
	WorkDir() string
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

// SaveFile is a no-op for local storage beyond deriving the key: processors
// already write their output into the base dir, so the produced filename is
// the key.
func (s *LocalStore) SaveFile(_ context.Context, localPath string) (string, error) {
	if filepath.Dir(localPath) == filepath.Clean(s.baseDir) {
		return filepath.Base(localPath), nil
	}

	// Written somewhere else (a scratch dir): move it into the store.
	key := uuid.New().String() + filepath.Ext(localPath)
	data, err := os.ReadFile(localPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(s.baseDir, key), data, 0o644); err != nil {
		return "", err
	}
	return key, nil
}

func (s *LocalStore) Fetch(_ context.Context, key string) (string, func(), error) {
	return filepath.Join(s.baseDir, key), func() {}, nil
}

// WorkDir is where processors should write their output. For local storage
// that's the store itself, which is why SaveFile usually costs nothing.
func (s *LocalStore) WorkDir() string { return s.baseDir }

func (s *LocalStore) Delete(_ context.Context, key string) error {
	return os.Remove(s.LocalPath(key))
}

// LocalPath is retained for local-only callers (the ZIP builder reads
// straight off disk). Anything that must also work against S3 uses Fetch.
func (s *LocalStore) LocalPath(key string) string {
	return filepath.Join(s.baseDir, key)
}

// BaseDir exposes the directory processors write outputs into.
func (s *LocalStore) BaseDir() string { return s.baseDir }
