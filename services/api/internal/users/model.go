package users

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                  uuid.UUID
	Email               string
	PasswordHash        string
	DisplayName         string
	AvatarKey           *string
	ResetToken          *string
	ResetTokenExpiresAt *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
