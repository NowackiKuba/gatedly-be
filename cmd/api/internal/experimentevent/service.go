package experimentevent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
	"toggly.com/m/pkg/response"
)

type Service interface {
	Create(ctx context.Context, e *domain.ExperimentEvent) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ExperimentEvent, error)
	GetByExperimentID(ctx context.Context, filters Filters, experimentID uuid.UUID) (*pagination.Page[domain.ExperimentEvent], error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, e *domain.ExperimentEvent) error {
	if err := s.repo.Create(ctx, e); err != nil {
		return fmt.Errorf("create experiment event: %w", err)
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*domain.ExperimentEvent, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get experiment event: %w", err)
	}
	if e == nil {
		return nil, response.NotFound("experiment event not found")
	}
	return e, nil
}

func (s *service) GetByExperimentID(ctx context.Context, filters Filters, experimentID uuid.UUID) (*pagination.Page[domain.ExperimentEvent], error) {
	page, err := s.repo.GetByExperimentID(ctx, filters, experimentID)
	if err != nil {
		return nil, fmt.Errorf("list experiment events: %w", err)
	}
	return page, nil
}
