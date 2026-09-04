package users

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/auth"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

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
