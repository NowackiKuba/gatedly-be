package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
)

type Filters struct {
	Limit        int    `json:"limit"`
	Offset       int    `json:"offset"`
	OrderBy      string `json:"orderBy"`
	OrderByField string `json:"orderByField"`
}

type Repository interface {
	Create(ctx context.Context, k *domain.APIKey) error
	ListByEnvironment(ctx context.Context, filters Filters, environmentID uuid.UUID) (*pagination.Page[domain.APIKey], error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error)
	GetByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, k *domain.APIKey) error {
	if err := r.db.WithContext(ctx).Create(k).Error; err != nil {
		return fmt.Errorf("apikey create: %w", err)
	}
	return nil
}

func (r *repository) ListByEnvironment(ctx context.Context, filters Filters, environmentID uuid.UUID) (*pagination.Page[domain.APIKey], error) {
	var list []domain.APIKey

	orderField := filters.OrderByField
	if orderField == "" {
		orderField = "id"
	}
	order := filters.OrderBy
	if order == "" {
		order = "asc"
	}
	orderParam := orderField + " " + order

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&domain.APIKey{}).
		Where("environment_id = ?", environmentID).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("apikey list by environment: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("environment_id = ?", environmentID).
		Order(orderParam).
		Limit(filters.Limit).
		Offset(filters.Offset).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("apikey list by environment: %w", err)
	}

	page := pagination.Paginate(list, filters.Limit, filters.Offset, int(total))
	return &page, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	var k domain.APIKey
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&k).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("apikey get by id: %w", err)
	}
	return &k, nil
}

func (r *repository) GetByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error) {
	var k domain.APIKey
	err := r.db.WithContext(ctx).Where("prefix = ?", prefix).First(&k).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("apikey get by prefix: %w", err)
	}
	return &k, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.APIKey{}).Error; err != nil {
		return fmt.Errorf("apikey delete: %w", err)
	}
	return nil
}

func (r *repository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&domain.APIKey{}).Where("id = ?", id).Update("last_used_at", now).Error; err != nil {
		return fmt.Errorf("apikey update last used: %w", err)
	}
	return nil
}
