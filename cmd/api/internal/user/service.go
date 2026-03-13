package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/response"
)

// Service handles user business logic.
type Service struct {
	repo *Repository
}

// NewService returns a new user service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetByID returns the user by ID or NotFound error.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return nil, response.NotFound("user not found")
	}
	return u, nil
}

// GetByEmail returns the user by email or nil if not found.
func (s *Service) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// UpdateProfile updates the user's name. Returns updated user or error.
func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, name string) (*domain.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return nil, response.NotFound("user not found")
	}
	u.Name = name
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return u, nil
}
