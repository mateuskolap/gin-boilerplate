package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"

	"uuid"
)

const (
	RoleAdmin = "Admin"
	RoleUser  = "User"
)

type Role struct {
	shared.BaseModel
	Name string `json:"name" gorm:"not null;unique"`

	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;constraint:OnDelete:CASCADE;"`
	Users       []User       `json:"users,omitempty" gorm:"many2many:user_roles;constraint:OnDelete:CASCADE;"`
}

type RoleRepository interface {
	shared.BaseRepository[Role]

	// GetByName finds a role by its unique name, optionally preloading relationships.
	// Returns (nil, nil) if no role matches the name.
	GetByName(ctx context.Context, name string, preloads ...string) (*Role, error)

	// AddPermissions adds permissions to a role.
	AddPermissions(ctx context.Context, role Role, permissionIDs []uuid.UUID) error

	// RemovePermissions removes permissions from a role.
	RemovePermissions(ctx context.Context, role Role, permissionIDs []uuid.UUID) error
}

type RoleUseCase interface {
	shared.BaseListUseCase[Role]

	shared.BaseFindUseCase[Role]

	shared.BaseDeleteUseCase

	// Create creates a new role.
	Create(ctx context.Context, role *Role) error

	// Update updates an existing role.
	Update(ctx context.Context, role *Role) error

	// AddPermissions adds permissions to a role.
	AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error

	// RemovePermissions removes permissions from a role.
	RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
}
