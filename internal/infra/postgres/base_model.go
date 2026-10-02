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

func BaseModelFromDomain(domainModel shared.BaseModel) BaseModel {
	return BaseModel{
		ID:        domainModel.ID,
		CreatedAt: domainModel.CreatedAt,
		UpdatedAt: domainModel.UpdatedAt,
	}
}

func (m BaseModel) ToDomain() shared.BaseModel {
	return shared.BaseModel{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
