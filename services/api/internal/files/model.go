package files

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID                uuid.UUID
	OwnerID           uuid.UUID
	Filename          string
	MimeType          string
	Size              int64
	Checksum          string
	StorageKey        string
	DerivedFromFileID *uuid.UUID
	CreatedAt         time.Time
}
