package environment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
	"toggly.com/m/pkg/response"
)

type Service interface {
	Create(ctx context.Context, env *domain.Environment) error
	Update(ctx context.Context, env *domain.Environment) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Environment, error)
	GetByProjectId(ctx context.Context, id uuid.UUID) (*pagination.Page[domain.Environment], error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, env *domain.Environment) error {
	exixts := s.repo.ExistsByProjectIdAndSlug(ctx, env.ProjectID, env.Slug)

	if exixts {
		// TODO

	}
	if err := s.repo.Create(ctx, env); err != nil {
		return fmt.Errorf("create environment: %w", err)
	}
	return nil
}

func (s *service) Update(ctx context.Context, env *domain.Environment) error {
	existing, err := s.repo.GetById(ctx, env.ID)
	if err != nil {
		return fmt.Errorf("get environment: %w", err)
	}
	if existing == nil {
		return response.NotFound("environment not found")
	}
	if err := s.repo.Update(ctx, env); err != nil {
		return fmt.Errorf("update environment: %w", err)
	}
	return nil
}

func (s *service) GetById(ctx context.Context, id uuid.UUID) (*domain.Environment, error) {
	env, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get environment: %w", err)
	}
	if env == nil {
		return nil, response.NotFound("environment not found")
	}
	return env, nil
}

func (s *service) GetByProjectId(ctx context.Context, id uuid.UUID) (*pagination.Page[domain.Environment], error) {
	page, err := s.repo.GetByProjectId(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get environments by project: %w", err)
	}
	return page, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetById(ctx, id)
	if err != nil {
		return fmt.Errorf("get environment: %w", err)
	}
	if existing == nil {
		return response.NotFound("environment not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	return nil
}
