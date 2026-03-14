package apikey

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/crypto"
	"toggly.com/m/pkg/pagination"
	"toggly.com/m/pkg/response"
)

type Service interface {
	Generate(ctx context.Context, environmentID uuid.UUID, name string) (*domain.APIKey, string, error) // APIKey + plaintext key (only once)
	Verify(ctx context.Context, rawKey string) (*domain.APIKey, error)
	List(ctx context.Context, filters Filters, environmentID uuid.UUID) (*pagination.Page[domain.APIKey], error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// buildPrefix returns "sk_" + first 5 chars of hex (8 chars total).
func buildPrefix(hexStr string) string {
	if len(hexStr) < 5 {
		return "sk_" + hexStr
	}
	return "sk_" + hexStr[:5]
}

// fullKey returns prefix + "_" + remaining hex.
func fullKey(prefix, hexStr string) string {
	if len(hexStr) <= 5 {
		return prefix
	}
	return prefix + "_" + hexStr[5:]
}

// extractPrefix returns the prefix from a raw key (everything before the second "_").
func extractPrefix(rawKey string) string {
	parts := strings.SplitN(rawKey, "_", 3) // "sk", "a1b2c", "rest..."
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "_" + parts[1]
}

func (s *service) Generate(ctx context.Context, environmentID uuid.UUID, name string) (*domain.APIKey, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", response.BadRequest("name is required")
	}

	hexStr, err := crypto.RandomToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("apikey random token: %w", err)
	}
	prefix := buildPrefix(hexStr)
	full := fullKey(prefix, hexStr)

	keyHash, err := crypto.HashPassword(full)
	if err != nil {
		return nil, "", fmt.Errorf("apikey hash: %w", err)
	}

	k := &domain.APIKey{
		EnvironmentID: environmentID,
		Name:          name,
		Prefix:        prefix,
		KeyHash:       keyHash,
	}
	if err := s.repo.Create(ctx, k); err != nil {
		return nil, "", fmt.Errorf("apikey create: %w", err)
	}
	return k, full, nil
}

func (s *service) Verify(ctx context.Context, rawKey string) (*domain.APIKey, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return nil, response.Unauthorized("invalid api key")
	}
	prefix := extractPrefix(rawKey)
	if prefix == "" {
		return nil, response.Unauthorized("invalid api key")
	}

	k, err := s.repo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("apikey get by prefix: %w", err)
	}
	if k == nil {
		return nil, response.Unauthorized("invalid api key")
	}

	if !crypto.CheckPassword(k.KeyHash, rawKey) {
		return nil, response.Unauthorized("invalid api key")
	}

	if err := s.repo.UpdateLastUsed(ctx, k.ID); err != nil {
		// non-fatal: still return the key
	}
	return k, nil
}

func (s *service) List(ctx context.Context, filters Filters, environmentID uuid.UUID) (*pagination.Page[domain.APIKey], error) {
	page, err := s.repo.ListByEnvironment(ctx, filters, environmentID)
	if err != nil {
		return nil, fmt.Errorf("apikey list: %w", err)
	}
	return page, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("apikey get: %w", err)
	}
	if existing == nil {
		return response.NotFound("api key not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("apikey delete: %w", err)
	}
	return nil
}
