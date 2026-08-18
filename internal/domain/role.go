package domain

import (
	"context"

	"github.com/google/uuid"
)

type Role struct {
	BaseModel
	Name string `json:"name" gorm:"not null;unique"`
}

type RoleRepository interface {
	BaseRepository[Role]
	GetByName(ctx context.Context, name string) (*Role, error)
}

type RoleUseCase interface {
	Find(ctx context.Context, id uuid.UUID)
	Create(ctx context.Context, role *Role) error
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}
