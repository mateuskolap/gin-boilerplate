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

	// GetByName finds a role by its unique name.
	// Returns (nil, nil) if no role matches the name.
	GetByName(ctx context.Context, name string) (*Role, error)
}

type RoleUseCase interface {
	// Find retrieves a role by its UUID.
	Find(ctx context.Context, id uuid.UUID) (*Role, error)

	// Create creates a new role.
	Create(ctx context.Context, role *Role) error

	// Update updates an existing role.
	Update(ctx context.Context, role *Role) error

	// Delete removes a role by its UUID.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves a paginated collection of roles.
	List(ctx context.Context, params PaginationParams, filters []Filter) (*PaginatedResult[Role], error)
}
