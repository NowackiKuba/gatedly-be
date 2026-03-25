package packet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/response"
)

type Service interface {
	Create(ctx context.Context, p *domain.Packet) error
	Update(ctx context.Context, p *domain.Packet) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Packet, error)
	GetAll(ctx context.Context) (*[]domain.Packet, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, p *domain.Packet) error {
	if err := s.repo.Create(ctx, p); err != nil {
		return fmt.Errorf("packet create: %w", err)
	}
	return nil
}

func (s *service) Update(ctx context.Context, p *domain.Packet) error {
	if err := s.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("packet update: %w", err)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("packet delete: %w", err)
	}
	return nil
}

func (s *service) GetById(ctx context.Context, id uuid.UUID) (*domain.Packet, error) {
	p, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("packet get by id: %w", err)
	}
	if p == nil {
		return nil, response.NotFound("packet not found")
	}
	return p, nil
}

func (s *service) GetAll(ctx context.Context) (*[]domain.Packet, error) {
	list, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("packet get all: %w", err)
	}
	return list, nil
}
