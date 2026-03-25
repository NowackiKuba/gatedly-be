package packet

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, p *domain.Packet) error
	Update(ctx context.Context, p *domain.Packet) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Packet, error)
	GetAll(ctx context.Context) (*[]domain.Packet, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, p *domain.Packet) error {
	if err := r.db.WithContext(ctx).Create(&p).Error; err != nil {
		return err
	}

	return nil
}

func (r *repository) Update(ctx context.Context, p *domain.Packet) error {
	if err := r.db.WithContext(ctx).Save(&p).Error; err != nil {
		return err
	}

	return nil
}
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Packet{}).Error; err != nil {
		return err
	}

	return nil
}
func (r *repository) GetById(ctx context.Context, id uuid.UUID) (*domain.Packet, error) {
	var p domain.Packet

	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &p, nil
}

func (r *repository) GetAll(ctx context.Context) (*[]domain.Packet, error) {
	var list []domain.Packet

	if err := r.db.WithContext(ctx).Find(&list).Error; err != nil {
		return nil, err
	}

	return &list, nil
}
