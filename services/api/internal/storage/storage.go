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
	Save(ctx context.Context, ownerID uuid.UUID, data []byte, ext string) (key string, err error)
	SaveFile(ctx context.Context, ownerID uuid.UUID, localPath string) (key string, err error)
	Fetch(ctx context.Context, key string) (localPath string, cleanup func(), err error)
	WorkDir() string
	Delete(ctx context.Context, key string) error
}

// ObjectKey builds the storage key for a new object: one folder per owner,
// then an opaque UUID. The owner prefix makes a bucket browsable and
// auditable per user, and makes a "delete everything belonging to this
// account" a prefix operation rather than a scan.
//
// The filename half stays a UUID rather than the user's own filename - the
// original name lives in the database, so a key still leaks nothing about
// the content and can't be guessed. Only the *shape* of the key changed.
//
// Keys are stored in full in the database, so Fetch and Delete keep working
// unchanged against objects written before this prefix existed: an old
// unprefixed key round-trips exactly as it always did, and no backfill is
// required.
func ObjectKey(ownerID uuid.UUID, ext string) string {
	return path.Join("users", ownerID.String(), uuid.New().String()+ext)
}

// safeKey rejects a key that would escape the store's base directory. Keys
// are server-generated and come back from our own database, so this is
// defence in depth rather than a live threat - but keys contain separators
// now, which is exactly when traversal stops being impossible by accident.
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

// SaveFile moves a produced file into the store under its owner's prefix.
//
// This used to be free: processors write into WorkDir, which is the base
// dir, so the produced filename was already the key. With an owner prefix
// the file has to actually move into that folder, which on the same volume
// is a rename - still cheap, just no longer nothing. The copy fallback is
// for the case where WorkDir and the store end up on different filesystems,
// where rename fails with EXDEV.
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

// LocalPath is retained for local-only callers (the ZIP builder reads
// straight off disk). Anything that must also work against S3 uses Fetch.
//
// Keys use forward slashes on every platform (they are S3 keys first), so
// they are converted to the OS separator here rather than at each caller.
func (s *LocalStore) LocalPath(key string) string {
	return filepath.Join(s.baseDir, filepath.FromSlash(key))
}

// BaseDir exposes the directory processors write outputs into.
func (s *LocalStore) BaseDir() string { return s.baseDir }
