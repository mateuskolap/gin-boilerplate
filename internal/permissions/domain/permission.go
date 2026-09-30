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
	PermissionViewPermission  PermissionName = "view_permission"
	PermissionViewActivityLog PermissionName = "view_activity_log"

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
	PermissionViewActivityLog,
	PermissionViewUser,
	PermissionAddUserRole,
	PermissionRemoveUserRole,
	PermissionDeleteUser,
	PermissionUpdateUser,
}

type Permission struct {
	shared.BaseModel
	Name string `json:"name"`
}

type PermissionRepository interface {
	List(ctx context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[Permission], error)
}

type PermissionUseCase interface {
	shared.BaseListUseCase[Permission]
}
