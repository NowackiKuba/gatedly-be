package subscription

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
	"toggly.com/m/pkg/response"
)

type Service interface {
	Create(ctx context.Context, s *domain.Subscription) error
	Update(ctx context.Context, s *domain.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	GetByStripeID(ctx context.Context, stripeID string) (*domain.Subscription, error)
	List(ctx context.Context, filters Filters) (*pagination.Page[domain.Subscription], error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, sub *domain.Subscription) error {
	if err := s.repo.Create(ctx, sub); err != nil {
		return fmt.Errorf("subscription create: %w", err)
	}
	return nil
}

func (s *service) Update(ctx context.Context, sub *domain.Subscription) error {
	if err := s.repo.Update(ctx, sub); err != nil {
		return fmt.Errorf("subscription update: %w", err)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("subscription get: %w", err)
	}
	if existing == nil {
		return response.NotFound("subscription not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("subscription delete: %w", err)
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("subscription get by id: %w", err)
	}
	if sub == nil {
		return nil, response.NotFound("subscription not found")
	}
	return sub, nil
}

func (s *service) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("subscription get by user id: %w", err)
	}
	if sub == nil {
		return nil, response.NotFound("subscription not found")
	}
	return sub, nil
}

func (s *service) GetByStripeID(ctx context.Context, stripeID string) (*domain.Subscription, error) {
	sub, err := s.repo.GetByStripeID(ctx, stripeID)
	if err != nil {
		return nil, fmt.Errorf("subscription get by stripe id: %w", err)
	}
	if sub == nil {
		return nil, response.NotFound("subscription not found")
	}
	return sub, nil
}

func (s *service) List(ctx context.Context, filters Filters) (*pagination.Page[domain.Subscription], error) {
	page, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("subscription list: %w", err)
	}
	return page, nil
}
