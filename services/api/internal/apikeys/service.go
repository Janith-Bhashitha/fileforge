package apikeys

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const keyPrefix = "ffk_"
const prefixVisibleChars = 8 // "ffk_" + 8 hex chars = 12-char lookup prefix

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create generates a new key and returns both the row (for listing) and the
// raw key (shown to the caller exactly once - it's never recoverable again,
// only the bcrypt hash is kept).
func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, name string) (*APIKey, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}
	rawHex := hex.EncodeToString(raw)
	fullKey := keyPrefix + rawHex
	lookupPrefix := keyPrefix + rawHex[:prefixVisibleChars]

	hash, err := bcrypt.GenerateFromPassword([]byte(fullKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	k := &APIKey{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Name:      name,
		KeyPrefix: lookupPrefix,
		KeyHash:   string(hash),
	}
	if err := s.repo.Create(ctx, k); err != nil {
		return nil, "", err
	}

	return k, fullKey, nil
}

// Verify is the auth-path lookup: narrow by prefix, then bcrypt-compare
// against that handful of candidates (in practice 0 or 1). Touches
// last_used_at on success, best-effort - a failure there shouldn't fail
// the request it's just bookkeeping for.
func (s *Service) Verify(ctx context.Context, rawKey string) (*APIKey, error) {
	if len(rawKey) < len(keyPrefix)+prefixVisibleChars {
		return nil, fmt.Errorf("invalid api key format")
	}
	lookupPrefix := rawKey[:len(keyPrefix)+prefixVisibleChars]

	candidates, err := s.repo.CandidatesByPrefix(ctx, lookupPrefix)
	if err != nil {
		return nil, err
	}

	for _, k := range candidates {
		if bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(rawKey)) == nil {
			_ = s.repo.TouchLastUsed(ctx, k.ID)
			return &k, nil
		}
	}
	return nil, fmt.Errorf("invalid or revoked api key")
}

func (s *Service) List(ctx context.Context, ownerID uuid.UUID) ([]APIKey, error) {
	return s.repo.ListByOwner(ctx, ownerID)
}

func (s *Service) Revoke(ctx context.Context, id, ownerID uuid.UUID) error {
	return s.repo.Revoke(ctx, id, ownerID)
}
