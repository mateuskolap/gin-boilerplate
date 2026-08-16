package repository

import (
	"context"
	"errors"
	"gin-boilerplate/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type baseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) domain.BaseRepository[T] {
	return &baseRepository[T]{
		db: db,
	}
}

func (r *baseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *baseRepository[T]) GetByID(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &entity, nil
}

func (r *baseRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *baseRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	var entity T
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity).Error
}
