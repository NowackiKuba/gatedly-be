package flag

import (
	"context"
	"fmt"

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
	Create(ctx context.Context, f *domain.Flag) error
	Update(ctx context.Context, f *domain.Flag) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Flag, error)
	GetByKey(ctx context.Context, key string) (*domain.Flag, error)
	GetByProjectIdAndKey(ctx context.Context, projectID uuid.UUID, key string) (*domain.Flag, error)
	GetByProjectId(ctx context.Context, filters Filters, pID uuid.UUID) (*pagination.Page[domain.Flag], error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, f *domain.Flag) error {
	if err := r.db.WithContext(ctx).Create(&f).Error; err != nil {
		return err
	}

	return nil
}
func (r *repository) Update(ctx context.Context, f *domain.Flag) error {
	if err := r.db.WithContext(ctx).Save(&f).Error; err != nil {
		return err
	}

	return nil
}
func (r *repository) GetById(ctx context.Context, id uuid.UUID) (*domain.Flag, error) {
	var f domain.Flag
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&f).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *repository) GetByKey(ctx context.Context, key string) (*domain.Flag, error) {
	var f domain.Flag
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&f).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *repository) GetByProjectIdAndKey(ctx context.Context, projectID uuid.UUID, key string) (*domain.Flag, error) {
	var f domain.Flag
	err := r.db.WithContext(ctx).Where("project_id = ? AND key = ?", projectID, key).First(&f).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *repository) GetByProjectId(ctx context.Context, filters Filters, pID uuid.UUID) (*pagination.Page[domain.Flag], error) {
	var list []domain.Flag

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
		Model(&domain.Flag{}).
		Where("project_id = ?", pID).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("flag get by project id: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("project_id = ?", pID).
		Order(orderParam).
		Limit(filters.Limit).
		Offset(filters.Offset).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("flag get by project id: %w", err)
	}

	page := pagination.Paginate(list, filters.Limit, filters.Offset, int(total))
	return &page, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Flag{}).Error; err != nil {
		return fmt.Errorf("flag delete: %w", err)
	}
	return nil
}
