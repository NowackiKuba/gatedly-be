package flag

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
	"toggly.com/m/pkg/response"
)

type Service interface {
	Create(ctx context.Context, f *domain.Flag) error
	Update(ctx context.Context, f *domain.Flag) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Flag, error)
	GetByKey(ctx context.Context, key string) (*domain.Flag, error)
	GetByProjectId(ctx context.Context, filters Filters, projectID uuid.UUID) (*pagination.Page[domain.Flag], error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, f *domain.Flag) error {
	existing, err := s.repo.GetByProjectIdAndKey(ctx, f.ProjectID, f.Key)
	if err != nil {
		return fmt.Errorf("check flag key: %w", err)
	}
	if existing != nil {
		return response.Conflict("flag with this key already exists in the project")
	}
	if err := s.repo.Create(ctx, f); err != nil {
		return fmt.Errorf("create flag: %w", err)
	}
	return nil
}

func (s *service) Update(ctx context.Context, f *domain.Flag) error {
	existing, err := s.repo.GetById(ctx, f.ID)
	if err != nil {
		return fmt.Errorf("get flag: %w", err)
	}
	if existing == nil {
		return response.NotFound("flag not found")
	}
	if f.Key != "" && f.Key != existing.Key {
		byKey, err := s.repo.GetByProjectIdAndKey(ctx, f.ProjectID, f.Key)
		if err != nil {
			return fmt.Errorf("check flag key: %w", err)
		}
		if byKey != nil {
			return response.Conflict("flag with this key already exists in the project")
		}
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return fmt.Errorf("update flag: %w", err)
	}
	return nil
}

func (s *service) GetById(ctx context.Context, id uuid.UUID) (*domain.Flag, error) {
	f, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}
	if f == nil {
		return nil, response.NotFound("flag not found")
	}
	return f, nil
}

func (s *service) GetByKey(ctx context.Context, key string) (*domain.Flag, error) {
	f, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get flag by key: %w", err)
	}
	if f == nil {
		return nil, response.NotFound("flag not found")
	}
	return f, nil
}

func (s *service) GetByProjectId(ctx context.Context, filters Filters, projectID uuid.UUID) (*pagination.Page[domain.Flag], error) {
	page, err := s.repo.GetByProjectId(ctx, filters, projectID)
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}
	return page, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetById(ctx, id)
	if err != nil {
		return fmt.Errorf("get flag: %w", err)
	}
	if existing == nil {
		return response.NotFound("flag not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete flag: %w", err)
	}
	return nil
}
