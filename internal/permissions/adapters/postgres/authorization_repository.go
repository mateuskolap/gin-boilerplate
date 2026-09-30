package postgres

import (
	"context"

	postgresinfra "gin-boilerplate/internal/infra/postgres"
	permissiondomain "gin-boilerplate/internal/permissions/domain"

	"gorm.io/gorm"
	"uuid"
)

type authorizationRepository struct {
	db *gorm.DB
}

func NewAuthorizationRepository(db *gorm.DB) permissiondomain.AuthorizationRepository {
	return &authorizationRepository{db: db}
}

func (r *authorizationRepository) UserHasPermission(
	ctx context.Context,
	userID uuid.UUID,
	permission permissiondomain.PermissionName,
) (bool, error) {
	var matchedPermission PermissionModel

	result := postgresinfra.GetTxFromContext(ctx, r.db).
		Model(&PermissionModel{}).
		Select("permissions.id").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Joins("JOIN users ON users.id = user_roles.user_id").
		Where("users.id = ? AND users.deleted_at IS NULL AND permissions.name = ?", userID, permission).
		Limit(1).
		Find(&matchedPermission)

	return result.RowsAffected > 0, result.Error
}
