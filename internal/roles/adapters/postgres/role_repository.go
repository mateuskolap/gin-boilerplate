package postgres

import (
	"context"
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	permissionpostgres "gin-boilerplate/internal/permissions/adapters/postgres"
	roledomain "gin-boilerplate/internal/roles/domain"

	"uuid"

	"gorm.io/gorm"
)

type roleRepository struct {
	*postgresinfra.ActivityLoggingRepository[roledomain.Role]
}

func NewRoleRepository(db *gorm.DB, activityLogRepo activitylogdomain.ActivityLogRepository) roledomain.RoleRepository {
	return &roleRepository{
		ActivityLoggingRepository: postgresinfra.NewActivityLoggingRepository(
			db,
			postgresinfra.NewBaseRepository(db,
				func() any { return &RoleModel{} },
				func(role *roledomain.Role) any { return roleModelFromDomain(role) },
				roleDomainFromModel,
			),
			activityLogRepo,
		),
	}
}

func (r *roleRepository) GetByIDWithPermissions(ctx context.Context, id uuid.UUID) (*roledomain.Role, error) {
	return r.FindOneBy(ctx, "id = ?", []any{id}, "Permissions")
}

func (r *roleRepository) GetByName(ctx context.Context, name string) (*roledomain.Role, error) {
	return r.FindOneBy(ctx, "name = ?", []any{name})
}

func (r *roleRepository) AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	permissions := make([]permissionpostgres.PermissionModel, len(permissionIDs))
	for i, id := range permissionIDs {
		permissions[i].ID = id
	}

	return r.DB(ctx).WithContext(ctx).
		Model(&RoleModel{ID: roleID}).
		Association("Permissions").Append(&permissions)
}

func (r *roleRepository) RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	permissions := make([]permissionpostgres.PermissionModel, len(permissionIDs))
	for i, id := range permissionIDs {
		permissions[i].ID = id
	}

	return r.DB(ctx).WithContext(ctx).
		Model(&RoleModel{ID: roleID}).
		Association("Permissions").Delete(&permissions)
}
