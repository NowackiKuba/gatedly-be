package project

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
)

type Filters struct {
	Limit        int64  `json:"limit"`
	Offset       int64  `json:"offset"`
	OrderBy      string `json:"orderBy"`
	OrderByField string `json:"orderByField"`
}

type Repository interface {
	Create(ctx context.Context, p *domain.Project) error
	Update(ctx context.Context, p *domain.Project) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)
	GetByUserId(ctx context.Context, filters Filters, userId uuid.UUID) (pagination.Page[domain.Project], error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, p *domain.Project) error {
	if err := r.db.WithContext(ctx).Create(&p).Error; err != nil {
		return err
	}

	return nil
}

func (r *repository) Update(ctx context.Context, p *domain.Project) error {
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}

	return nil
}

func (r *repository) GetById(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	var p domain.Project
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (*domain.Project, error) {
	var p domain.Project
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&p).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}
func (r *repository) GetByUserId(ctx context.Context, filters Filters, userId uuid.UUID) (pagination.Page[domain.Project], error) {
	var projects []domain.Project

	// Default to "id" if no field provided
	orderField := filters.OrderByField
	if orderField == "" {
		orderField = "id"
	}

	// Default to "asc" if no order provided
	order := filters.OrderBy
	if order == "" {
		order = "asc"
	}

	// Build full order string, e.g. "id asc"
	orderParam := orderField + " " + order

	// Fetch total count
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&domain.Project{}).
		Where("owner_id = ?", userId).
		Count(&total).Error; err != nil {
		return pagination.Page[domain.Project]{}, err
	}

	// Fetch paginated results
	if err := r.db.WithContext(ctx).
		Where("owner_id = ?", userId).
		Order(orderParam).
		Limit(int(filters.Limit)).
		Offset(int(filters.Offset)).
		Find(&projects).Error; err != nil {
		return pagination.Page[domain.Project]{}, err
	}

	page := pagination.Paginate[domain.Project](projects, int(filters.Limit), int(filters.Offset), int(total))

	return page, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Project{}).Error; err != nil {
		return err
	}

	return nil
}
