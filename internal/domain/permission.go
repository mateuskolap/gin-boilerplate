package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
)

type PermissionName string

const (
	// Roles
	PermissionCreateRole           PermissionName = "create_role"
	PermissionUpdateRole           PermissionName = "update_role"
	PermissionDeleteRole           PermissionName = "delete_role"
	PermissionViewRole             PermissionName = "view_role"
	PermissionAddRolePermission    PermissionName = "add_role_permission"
	PermissionRemoveRolePermission PermissionName = "remove_role_permission"

	// Permissions
	PermissionViewPermission PermissionName = "view_permission"

	// Users
	PermissionViewUser       PermissionName = "view_user"
	PermissionAddUserRole    PermissionName = "add_user_role"
	PermissionRemoveUserRole PermissionName = "remove_user_role"
	PermissionDeleteUser     PermissionName = "delete_user"
	PermissionUpdateUser     PermissionName = "update_user"
)

var AllPermissions = []PermissionName{
	PermissionCreateRole,
	PermissionUpdateRole,
	PermissionDeleteRole,
	PermissionViewRole,
	PermissionAddRolePermission,
	PermissionRemoveRolePermission,
	PermissionViewPermission,
	PermissionViewUser,
	PermissionAddUserRole,
	PermissionRemoveUserRole,
	PermissionDeleteUser,
	PermissionUpdateUser,
}

type Permission struct {
	shared.BaseModel
	Name string `json:"name" gorm:"unique;not null"`

	Roles []Role `json:"roles,omitempty" gorm:"many2many:role_permissions;constraint:OnDelete:CASCADE;"`
}

type PermissionRepository interface {
	shared.BaseRepository[Permission]

	// UpsertByName inserts or updates permissions based on their names.
	UpsertByName(ctx context.Context, permissions []Permission) error
}

type PermissionUseCase interface {
	shared.BaseListUseCase[Permission]

	// Find populates the permissions table with the predefined permissions.
	SeedPermissions(ctx context.Context) error
}
