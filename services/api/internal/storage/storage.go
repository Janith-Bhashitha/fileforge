package storage

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Store persists file bytes and hands back an opaque key.
//
// The interface is path-based rather than stream-based because LibreOffice,
// pdfcpu and pdftoppm all take paths. Fetch puts a key on local disk and
// returns the cleanup that releases it; Save and SaveFile put bytes back.
type Store interface {
	Save(ctx context.Context, ownerID uuid.UUID, data []byte, ext string) (key string, err error)
	SaveFile(ctx context.Context, ownerID uuid.UUID, localPath string) (key string, err error)
	Fetch(ctx context.Context, key string) (localPath string, cleanup func(), err error)
	WorkDir() string
	Delete(ctx context.Context, key string) error
}

// ObjectKey builds the key for a new object: one folder per owner, then an
// opaque UUID. The prefix makes per-account deletion a prefix operation
// rather than a scan; the UUID keeps the user's filename out of the key.
//
// Keys are stored in full in the database, so objects written before this
// prefix existed still resolve and need no backfill.
func ObjectKey(ownerID uuid.UUID, ext string) string {
	return path.Join("users", ownerID.String(), uuid.New().String()+ext)
}

// safeKey rejects a key that would escape the base directory. Keys are
// server-generated, but they contain separators now, so traversal is no
// longer impossible by construction.
func safeKey(key string) error {
	cleaned := path.Clean("/" + filepath.ToSlash(key))
	if cleaned == "/" || strings.Contains(key, "..") {
		return fmt.Errorf("invalid storage key %q", key)
	}
	return nil
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

func (s *LocalStore) Save(_ context.Context, ownerID uuid.UUID, data []byte, ext string) (string, error) {
	key := ObjectKey(ownerID, ext)
	dest := filepath.Join(s.baseDir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", err
	}
	return key, nil
}

// SaveFile moves a produced file under its owner's prefix. The copy is a
// fallback for when WorkDir and the store are on different filesystems and
// rename fails with EXDEV.
func (s *LocalStore) SaveFile(_ context.Context, ownerID uuid.UUID, localPath string) (string, error) {
	key := ObjectKey(ownerID, filepath.Ext(localPath))
	dest := filepath.Join(s.baseDir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}

	if err := os.Rename(localPath, dest); err == nil {
		return key, nil
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", err
	}
	os.Remove(localPath)
	return key, nil
}

func (s *LocalStore) Fetch(_ context.Context, key string) (string, func(), error) {
	if err := safeKey(key); err != nil {
		return "", func() {}, err
	}
	return s.LocalPath(key), func() {}, nil
}

// WorkDir is where processors should write their output. For local storage
// that's the store itself, which is why SaveFile usually costs nothing.
func (s *LocalStore) WorkDir() string { return s.baseDir }

func (s *LocalStore) Delete(_ context.Context, key string) error {
	if err := safeKey(key); err != nil {
		return err
	}
	return os.Remove(s.LocalPath(key))
}

// LocalPath is for local-only callers (the ZIP builder reads straight off
// disk). Anything that must also work against S3 uses Fetch. Keys are S3
// keys first, so the separator is converted here rather than at each caller.
func (s *LocalStore) LocalPath(key string) string {
	return filepath.Join(s.baseDir, filepath.FromSlash(key))
}

// BaseDir exposes the directory processors write outputs into.
func (s *LocalStore) BaseDir() string { return s.baseDir }
