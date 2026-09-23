package repository

import (
	"context"

	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"uuid"
)

type authorizationRepository struct {
	db *gorm.DB
}

func NewAuthorizationRepository(db *gorm.DB) domain.AuthorizationRepository {
	return &authorizationRepository{db: db}
}

func (r *authorizationRepository) UserHasPermission(
	ctx context.Context,
	userID uuid.UUID,
	permission domain.PermissionName,
) (bool, error) {
	var matchedPermission domain.Permission

	result := GetTxFromContext(ctx, r.db).
		Model(&domain.Permission{}).
		Select("permissions.id").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Joins("JOIN users ON users.id = user_roles.user_id").
		Where(clause.And(
			clause.Eq{
				Column: clause.Column{Table: "users", Name: "id"},
				Value:  userID,
			},
			clause.Eq{
				Column: clause.Column{Table: "users", Name: "deleted_at"},
				Value:  nil,
			},
			clause.Eq{
				Column: clause.Column{Table: "permissions", Name: "name"},
				Value:  permission,
			},
		)).
		Limit(1).
		Find(&matchedPermission)

	return result.RowsAffected > 0, result.Error
}
