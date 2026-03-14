package flagrule

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
	"toggly.com/m/pkg/response"
)

// CacheInvalidator is called after rule create/update/delete so evaluation cache can reload (e.g. evaluation.InvalidateCache).
// Pass nil to disable; avoids import cycle.
type CacheInvalidator func()

type Service interface {
	Create(ctx context.Context, r *domain.FlagRule) error
	Update(ctx context.Context, r *domain.FlagRule) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.FlagRule, error)
	GetByFlagId(ctx context.Context, filters Filters, flagID uuid.UUID) (*pagination.Page[domain.FlagRule], error)
	GetAll(ctx context.Context, flagID uuid.UUID) ([]domain.FlagRule, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo            Repository
	invalidateCache CacheInvalidator
}

func NewService(repo Repository, invalidateCache CacheInvalidator) Service {
	return &service{repo: repo, invalidateCache: invalidateCache}
}

func (s *service) Create(ctx context.Context, r *domain.FlagRule) error {
	if r.RolloutPct < 0 || r.RolloutPct > 100 {
		return response.BadRequest("rolloutPct must be between 0 and 100")
	}
	existing, err := s.repo.GetByFlagIdAndEnvironmentId(ctx, r.FlagID, r.EnvironmentID)
	if err != nil {
		return fmt.Errorf("check flag rule: %w", err)
	}
	if existing != nil {
		return response.Conflict("rule for this flag and environment already exists")
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return fmt.Errorf("create flag rule: %w", err)
	}
	if s.invalidateCache != nil {
		s.invalidateCache()
	}
	return nil
}

func (s *service) Update(ctx context.Context, r *domain.FlagRule) error {
	if r.RolloutPct < 0 || r.RolloutPct > 100 {
		return response.BadRequest("rolloutPct must be between 0 and 100")
	}
	existing, err := s.repo.GetById(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("get flag rule: %w", err)
	}
	if existing == nil {
		return response.NotFound("flag rule not found")
	}
	if err := s.repo.Update(ctx, r); err != nil {
		return fmt.Errorf("update flag rule: %w", err)
	}
	if s.invalidateCache != nil {
		s.invalidateCache()
	}
	return nil
}

func (s *service) GetById(ctx context.Context, id uuid.UUID) (*domain.FlagRule, error) {
	rule, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get flag rule: %w", err)
	}
	if rule == nil {
		return nil, response.NotFound("flag rule not found")
	}
	return rule, nil
}

func (s *service) GetByFlagId(ctx context.Context, filters Filters, flagID uuid.UUID) (*pagination.Page[domain.FlagRule], error) {
	page, err := s.repo.GetByFlagId(ctx, filters, flagID)
	if err != nil {
		return nil, fmt.Errorf("list flag rules: %w", err)
	}
	return page, nil
}

func (s *service) GetAll(ctx context.Context, flagID uuid.UUID) ([]domain.FlagRule, error) {
	list, err := s.repo.GetAllByFlagId(ctx, flagID)
	if err != nil {
		return nil, fmt.Errorf("get all flag rules: %w", err)
	}
	return list, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetById(ctx, id)
	if err != nil {
		return fmt.Errorf("get flag rule: %w", err)
	}
	if existing == nil {
		return response.NotFound("flag rule not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete flag rule: %w", err)
	}
	if s.invalidateCache != nil {
		s.invalidateCache()
	}
	return nil
}
