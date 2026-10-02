package postgres

import (
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	permissiondomain "gin-boilerplate/internal/permissions/domain"

	"gorm.io/gorm"
)

func NewPermissionRepository(db *gorm.DB) permissiondomain.PermissionRepository {
	return postgresinfra.NewBaseRepository(db,
		func() *PermissionModel { return &PermissionModel{} },
		func(permission *permissiondomain.Permission) *PermissionModel {
			return permissionModelFromDomain(permission)
		},
		permissionDomainFromModel,
	)
}
