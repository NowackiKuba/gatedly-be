package subscription

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
	Create(ctx context.Context, s *domain.Subscription) error
	Update(ctx context.Context, s *domain.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	GetByStripeID(ctx context.Context, stripeID string) (*domain.Subscription, error)
	List(ctx context.Context, filters Filters) (*pagination.Page[domain.Subscription], error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, s *domain.Subscription) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("subscription create: %w", err)
	}
	return nil
}

func (r *repository) Update(ctx context.Context, s *domain.Subscription) error {
	if err := r.db.WithContext(ctx).Save(s).Error; err != nil {
		return fmt.Errorf("subscription update: %w", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Subscription{}).Error; err != nil {
		return fmt.Errorf("subscription delete: %w", err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	var s domain.Subscription
	err := r.db.WithContext(ctx).Preload("Packet").Where("id = ?", id).First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription get by id: %w", err)
	}
	return &s, nil
}

func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	var s domain.Subscription
	err := r.db.WithContext(ctx).Preload("Packet").Where("user_id = ?", userID).First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription get by user id: %w", err)
	}
	return &s, nil
}

func (r *repository) GetByStripeID(ctx context.Context, stripeID string) (*domain.Subscription, error) {
	var s domain.Subscription
	err := r.db.WithContext(ctx).Preload("Packet").Where("stripe_id = ?", stripeID).First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("subscription get by stripe id: %w", err)
	}
	return &s, nil
}

func (r *repository) List(ctx context.Context, filters Filters) (*pagination.Page[domain.Subscription], error) {
	var list []domain.Subscription

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
	if err := r.db.WithContext(ctx).Model(&domain.Subscription{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("subscription list count: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Preload("Packet").
		Order(orderParam).
		Limit(filters.Limit).
		Offset(filters.Offset).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("subscription list: %w", err)
	}

	page := pagination.Paginate(list, filters.Limit, filters.Offset, int(total))
	return &page, nil
}
