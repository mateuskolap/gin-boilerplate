package postgres

import (
	"gin-boilerplate/internal/domain/shared"
	"time"

	"uuid"
)

type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func BaseModelFromDomain(entity shared.BaseModel) BaseModel {
	return BaseModel{
		ID:        entity.ID,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}

func (model BaseModel) ToDomain() shared.BaseModel {
	return shared.BaseModel{
		ID:        model.ID,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
