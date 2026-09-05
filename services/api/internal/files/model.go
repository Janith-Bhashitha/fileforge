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
	Operation         *string
	CreatedAt         time.Time
}

// ListItem is what the file list endpoint returns — it adds DerivedCount
// (how many outputs were produced from this file) which isn't part of the
// files table itself, just computed for display.
type ListItem struct {
	File
	DerivedCount int
}
