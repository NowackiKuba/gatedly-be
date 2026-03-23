package experimentevent

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/pkg/pagination"
)

type Filters struct {
	Limit        int                         `json:"limit"`
	Offset       int                         `json:"offset"`
	OrderBy      string                      `json:"orderBy"`
	OrderByField string                      `json:"orderByField"`
	EventType    *domain.ExperimentEventType `json:"eventType"`
}

type Repository interface {
	Create(ctx context.Context, e *domain.ExperimentEvent) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ExperimentEvent, error)
	GetByExperimentID(ctx context.Context, filters Filters, experimentID uuid.UUID) (*pagination.Page[domain.ExperimentEvent], error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, e *domain.ExperimentEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ExperimentEvent, error) {
	var e domain.ExperimentEvent
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *repository) GetByExperimentID(ctx context.Context, filters Filters, experimentID uuid.UUID) (*pagination.Page[domain.ExperimentEvent], error) {
	var list []domain.ExperimentEvent

	orderByField := filters.OrderByField
	if orderByField == "" {
		orderByField = "created_at"
	}

	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "desc"
	}

	orderParam := orderByField + " " + orderBy

	query := r.db.WithContext(ctx).Model(&domain.ExperimentEvent{}).Where("experiment_id = ?", experimentID)

	if filters.EventType != nil {
		query = query.Where("event_type = ?", *filters.EventType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.Order(orderParam).Limit(filters.Limit).Offset(filters.Offset).Find(&list).Error; err != nil {
		return nil, err
	}

	page := pagination.Paginate(list, filters.Limit, filters.Offset, int(total))
	return &page, nil
}
