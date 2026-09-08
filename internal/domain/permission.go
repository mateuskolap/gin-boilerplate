package domain

import (
	"context"
)

type PermissionName string

const (
	PermissionCreateRole PermissionName = "create_role"
	PermissionUpdateRole PermissionName = "update_role"
	PermissionDeleteRole PermissionName = "delete_role"
	PermissionViewRole   PermissionName = "view_role"
)

var AllPermissions = []PermissionName{
	PermissionCreateRole,
	PermissionUpdateRole,
	PermissionDeleteRole,
	PermissionViewRole,
}

type Permission struct {
	BaseModel
	Name string `json:"name" gorm:"unique;not null"`

	Roles []Role `json:"roles,omitempty" gorm:"many2many:role_permissions;constraint:OnDelete:CASCADE;"`
}

type PermissionRepository interface {
	BaseRepository[Permission]

	// UpsertByName inserts or updates permissions based on their names.
	UpsertByName(ctx context.Context, permissions []Permission) error

	// DeleteByNames removes permissions based on their names.
	DeleteByNames(ctx context.Context, names []string) error

	ListByRoleName(ctx context.Context, roleName string, params PaginationParams, filters []Filter) (*PaginatedResult[Permission], error)
}

type PermissionUseCase interface {
	BaseListUseCase[Permission]

	// Find populates the permissions table with the predefined permissions.
	SeedPermissions(ctx context.Context) error

	// FindByRole retrieves permissions associated with a specific role.
	ListByRoleName(ctx context.Context, roleName string, params PaginationParams, filters []Filter) (*PaginatedResult[Permission], error)
}
