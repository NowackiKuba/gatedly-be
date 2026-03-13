package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
)

// Repository handles user persistence.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a new user repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetByID returns the user by ID or nil if not found.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("user get by id: %w", err)
	}
	return &u, nil
}

// GetByEmail returns the user by email or nil if not found.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("user get by email: %w", err)
	}
	return &u, nil
}

// Create persists a new user.
func (r *Repository) Create(ctx context.Context, u *domain.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("user create: %w", err)
	}
	return nil
}

// Update saves the user (e.g. after profile update).
func (r *Repository) Update(ctx context.Context, u *domain.User) error {
	if err := r.db.WithContext(ctx).Save(u).Error; err != nil {
		return fmt.Errorf("user update: %w", err)
	}
	return nil
}
