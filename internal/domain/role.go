package domain

import (
	"context"

	"uuid"
)

type Role struct {
	BaseModel
	Name string `json:"name" gorm:"not null;unique"`

	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;constraint:OnDelete:CASCADE;"`
}

type RoleRepository interface {
	BaseRepository[Role]

	// GetByName finds a role by its unique name.
	// Returns (nil, nil) if no role matches the name.
	GetByName(ctx context.Context, name string) (*Role, error)

	// AddPermissions adds permissions to a role.
	AddPermissions(ctx context.Context, role Role, permissionIDs []uuid.UUID) error

	// RemovePermissions removes permissions from a role.
	RemovePermissions(ctx context.Context, role Role, permissionIDs []uuid.UUID) error
}

type RoleUseCase interface {
	BaseListUseCase[Role]

	BaseFindUseCase[Role]

	BaseDeleteUseCase

	// Create creates a new role.
	Create(ctx context.Context, role *Role) error

	// Update updates an existing role.
	Update(ctx context.Context, role *Role) error

	// AddPermissions adds permissions to a role.
	AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error

	// RemovePermissions removes permissions from a role.
	RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
}
