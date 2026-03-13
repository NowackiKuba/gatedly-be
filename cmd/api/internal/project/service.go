package project

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
	"toggly.com/m/pkg/response"
)

type Service interface {
	Create(ctx context.Context, p *domain.Project) error
	Update(ctx context.Context, p *domain.Project) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
	GetByUserId(ctx context.Context, filters Filters, userId uuid.UUID) (pagination.Page[domain.Project], error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, p *domain.Project) error {
	existing, err := s.repo.GetBySlug(ctx, p.Slug)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	if existing != nil {
		return response.Conflict("project with this slug already exists")
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

func (s *service) Update(ctx context.Context, p *domain.Project) error {
	existing, err := s.repo.GetById(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}
	if existing == nil {
		return response.NotFound("project not found")
	}

	// If slug changed, ensure uniqueness
	if p.Slug != "" && p.Slug != existing.Slug {
		bySlug, err := s.repo.GetBySlug(ctx, p.Slug)
		if err != nil {
			return fmt.Errorf("check slug uniqueness: %w", err)
		}
		if bySlug != nil && bySlug.ID != p.ID {
			return response.Conflict("project with this slug already exists")
		}
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

func (s *service) GetById(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	p, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if p == nil {
		return nil, response.NotFound("project not found")
	}
	return p, nil
}

func (s *service) GetBySlug(ctx context.Context, slug string) (*domain.Project, error) {
	p, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get project by slug: %w", err)
	}
	if p == nil {
		return nil, response.NotFound("project not found")
	}
	return p, nil
}

func (s *service) GetByUserId(ctx context.Context, filters Filters, userId uuid.UUID) (pagination.Page[domain.Project], error) {
	page, err := s.repo.GetByUserId(ctx, filters, userId)
	if err != nil {
		return pagination.Page[domain.Project]{}, fmt.Errorf("list projects: %w", err)
	}
	return page, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetById(ctx, id)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}
	if existing == nil {
		return response.NotFound("project not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}
