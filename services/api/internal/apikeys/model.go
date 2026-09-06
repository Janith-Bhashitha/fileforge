package apikeys

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	Name       string
	KeyPrefix  string
	KeyHash    string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}
