package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
)

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrInvalidResetToken = errors.New("invalid or expired reset link")
var ErrWrongCurrentPassword = errors.New("current password is incorrect")

const resetTokenTTL = time.Hour

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, email, password, displayName string) (*User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (*User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !auth.CheckPassword(password, u.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}

// UpdateProfile is the fix for display_name being write-once at
// registration with no way to ever change it.
func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, avatarKey *string) error {
	return s.repo.UpdateProfile(ctx, id, displayName, avatarKey)
}

// ChangePassword requires knowing the current password - distinct from the
// reset flow below, which is for someone who's locked out and can't prove
// that.
func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !auth.CheckPassword(currentPassword, u.PasswordHash) {
		return ErrWrongCurrentPassword
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordHash(ctx, id, hash)
}

// RequestPasswordReset always returns (user, nil) or (nil, nil) - never an
// error for "no such email". The handler must respond identically either
// way, or the endpoint becomes a way to check which emails have accounts.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (*User, string, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, "", nil
		}
		return nil, "", err
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", err
	}
	token := hex.EncodeToString(tokenBytes)

	if err := s.repo.SetResetToken(ctx, u.ID, token, time.Now().Add(resetTokenTTL)); err != nil {
		return nil, "", err
	}
	return u, token, nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	u, err := s.repo.GetByResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}
	if u.ResetTokenExpiresAt == nil || time.Now().After(*u.ResetTokenExpiresAt) {
		return ErrInvalidResetToken
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, u.ID, hash); err != nil {
		return err
	}
	return s.repo.ClearResetToken(ctx, u.ID)
}
