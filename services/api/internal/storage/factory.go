package storage

import (
	"context"
	"fmt"
	"path/filepath"
)

// Settings is the storage-relevant slice of the app config. It's declared
// here rather than taking *config.Config so the storage package stays
// importable without dragging config in behind it.
type Settings struct {
	Backend        string
	Dir            string
	Bucket         string
	Region         string
	Endpoint       string
	PublicEndpoint string
	ForcePathStyle bool
	AccessKeyID    string
	SecretKey      string
}

// New builds whichever Store the config asks for. Every binary (API and all
// three workers) calls this, so a deployment can't end up with the API on S3
// and a worker still reading local disk.
func New(ctx context.Context, s Settings) (Store, error) {
	switch s.Backend {
	case "", "local":
		return NewLocalStore(s.Dir)
	case "s3":
		return NewS3Store(ctx, S3Config{
			Bucket:          s.Bucket,
			Region:          s.Region,
			Endpoint:        s.Endpoint,
			PublicEndpoint:  s.PublicEndpoint,
			ForcePathStyle:  s.ForcePathStyle,
			AccessKeyID:     s.AccessKeyID,
			SecretAccessKey: s.SecretKey,
			WorkDir:         filepath.Join(s.Dir, "work"),
		})
	default:
		return nil, fmt.Errorf("unknown STORAGE_BACKEND %q (want \"local\" or \"s3\")", s.Backend)
	}
}
