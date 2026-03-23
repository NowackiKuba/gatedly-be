package experiments

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
	"toggly.com/m/pkg/response"
)

type Service interface {
	Create(ctx context.Context, e *domain.Experiment) error
	Update(ctx context.Context, e *domain.Experiment) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Experiment, error)
	GetByFlagID(ctx context.Context, filters Filters, flagID uuid.UUID) (*pagination.Page[domain.Experiment], error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, e *domain.Experiment) error {
	if err := validateVariants(e.Variants); err != nil {
		return err
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return fmt.Errorf("create experiment: %w", err)
	}
	return nil
}

func (s *service) Update(ctx context.Context, e *domain.Experiment) error {
	existing, err := s.repo.GetByID(ctx, e.ID)
	if err != nil {
		return fmt.Errorf("get experiment: %w", err)
	}
	if existing == nil {
		return response.NotFound("experiment not found")
	}
	if existing.Status == domain.ExperimentStatusCompleted {
		return response.BadRequest("cannot update a completed experiment")
	}
	if err := validateVariants(e.Variants); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return fmt.Errorf("update experiment: %w", err)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get experiment: %w", err)
	}
	if existing == nil {
		return response.NotFound("experiment not found")
	}
	if existing.Status == domain.ExperimentStatusRunning {
		return response.BadRequest("cannot delete a running experiment")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete experiment: %w", err)
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*domain.Experiment, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
	}
	if e == nil {
		return nil, response.NotFound("experiment not found")
	}
	return e, nil
}

func (s *service) GetByFlagID(ctx context.Context, filters Filters, flagID uuid.UUID) (*pagination.Page[domain.Experiment], error) {
	page, err := s.repo.GetByFlagID(ctx, filters, flagID)
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	return page, nil
}

func validateVariants(variants domain.ExperimentVariants) error {
	if len(variants) == 0 {
		return nil
	}
	if variants.TotalWeight() != 100 {
		return response.BadRequest("variant weights must sum to 100")
	}
	return nil
}
