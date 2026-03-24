package experiments

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
)

type Filters struct {
	Limit        int                     `json:"limit"`
	Offset       int                     `json:"offset"`
	OrderBy      string                  `json:"orderBy"`
	OrderByField string                  `json:"orderByField"`
	Status       domain.ExperimentStatus `json:"status"`
}

type Repository interface {
	Create(ctx context.Context, e *domain.Experiment) error
	Update(ctx context.Context, e *domain.Experiment) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Experiment, error)
	GetByFlagID(ctx context.Context, filters Filters, flagId, environmentId uuid.UUID) (*pagination.Page[domain.Experiment], error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, e *domain.Experiment) error {
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return err
	}

	return nil
}
func (r *repository) Update(ctx context.Context, e *domain.Experiment) error {
	if err := r.db.WithContext(ctx).Save(e).Error; err != nil {
		return err
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Experiment{}).Error; err != nil {
		return err
	}

	return nil
}
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Experiment, error) {
	var experiment domain.Experiment

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&experiment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &experiment, nil

}

func (r *repository) GetByFlagID(ctx context.Context, filters Filters, flagId, environmentId uuid.UUID) (*pagination.Page[domain.Experiment], error) {
	var list []domain.Experiment

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
		Model(&domain.Experiment{}).
		Where("flag_id = ? AND environment_id = ?", flagId, environmentId).
		Count(&total).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Where("flag_id = ? AND environment_id = ?", flagId, environmentId).
		Order(orderParam).
		Limit(filters.Limit).
		Offset(filters.Offset).
		Find(&list).Error; err != nil {
		return nil, err
	}

	page := pagination.Paginate(list, filters.Limit, filters.Offset, int(total))

	return &page, nil

}
