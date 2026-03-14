package environment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
)

type Filters struct {
	Limit        int
	Offset       int
	OrderBy      string
	OrderByField string
}

type Repository interface {
	Create(ctx context.Context, env *domain.Environment) error
	Update(ctx context.Context, env *domain.Environment) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Environment, error)
	GetByProjectId(ctx context.Context, filters Filters, projectID uuid.UUID) (*pagination.Page[domain.Environment], error)
	ExistsByProjectIdAndSlug(ctx context.Context, projectID uuid.UUID, slug string) bool
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, env *domain.Environment) error {
	if err := r.db.WithContext(ctx).Create(env).Error; err != nil {
		return fmt.Errorf("environment create: %w", err)
	}
	return nil
}

func (r *repository) Update(ctx context.Context, env *domain.Environment) error {
	if err := r.db.WithContext(ctx).Save(env).Error; err != nil {
		return fmt.Errorf("environment update: %w", err)
	}
	return nil
}

func (r *repository) GetById(ctx context.Context, id uuid.UUID) (*domain.Environment, error) {
	var env domain.Environment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&env).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("environment get by id: %w", err)
	}
	return &env, nil
}

func (r *repository) GetByProjectId(ctx context.Context, filters Filters, projectID uuid.UUID) (*pagination.Page[domain.Environment], error) {
	var list []domain.Environment
	orderField := filters.OrderByField
	if orderField == "" {
		orderField = "created_at"
	}
	order := filters.OrderBy
	if order == "" {
		order = "asc"
	}
	orderParam := orderField + " " + order

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&domain.Environment{}).
		Where("project_id = ?", projectID).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("environment count by project id: %w", err)
	}
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order(orderParam).
		Limit(filters.Limit).
		Offset(filters.Offset).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("environment get by project id: %w", err)
	}
	page := pagination.Paginate(list, filters.Limit, filters.Offset, int(total))
	return &page, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Environment{})
	if result.Error != nil {
		return fmt.Errorf("environment delete: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil
	}
	return nil
}

func (r *repository) ExistsByProjectIdAndSlug(ctx context.Context, projectID uuid.UUID, slug string) bool {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.Environment{}).
		Where("project_id = ? AND slug = ?", projectID, slug).
		Limit(1).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}
