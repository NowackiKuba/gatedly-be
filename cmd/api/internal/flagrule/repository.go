package flagrule

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, r *domain.FlagRule) error
	Update(ctx context.Context, r *domain.FlagRule) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.FlagRule, error)
	GetByFlagIdAndEnvironmentId(ctx context.Context, flagID, environmentID uuid.UUID) (*domain.FlagRule, error)
	GetByFlagId(ctx context.Context, flagID uuid.UUID) ([]domain.FlagRule, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, rule *domain.FlagRule) error {
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		return fmt.Errorf("flagrule create: %w", err)
	}
	return nil
}

func (r *repository) Update(ctx context.Context, rule *domain.FlagRule) error {
	if err := r.db.WithContext(ctx).Save(rule).Error; err != nil {
		return fmt.Errorf("flagrule update: %w", err)
	}
	return nil
}

func (r *repository) GetById(ctx context.Context, id uuid.UUID) (*domain.FlagRule, error) {
	var rule domain.FlagRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("flagrule get by id: %w", err)
	}
	return &rule, nil
}

func (r *repository) GetByFlagIdAndEnvironmentId(ctx context.Context, flagID, environmentID uuid.UUID) (*domain.FlagRule, error) {
	var rule domain.FlagRule
	err := r.db.WithContext(ctx).
		Where("flag_id = ? AND environment_id = ?", flagID, environmentID).
		First(&rule).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("flagrule get by flag and env: %w", err)
	}
	return &rule, nil
}

func (r *repository) GetByFlagId(ctx context.Context, flagID uuid.UUID) ([]domain.FlagRule, error) {
	var list []domain.FlagRule
	if err := r.db.WithContext(ctx).
		Where("flag_id = ?", flagID).
		Order("created_at ASC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("flagrule get by flag id: %w", err)
	}
	return list, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.FlagRule{}).Error; err != nil {
		return fmt.Errorf("flagrule delete: %w", err)
	}
	return nil
}
