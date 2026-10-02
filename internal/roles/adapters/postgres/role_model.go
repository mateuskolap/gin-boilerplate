package postgres

import (
	"gin-boilerplate/internal/infra/postgres"
	permissionpostgres "gin-boilerplate/internal/permissions/adapters/postgres"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
	roledomain "gin-boilerplate/internal/roles/domain"
)

type RoleModel struct {
	postgres.BaseModel
	Name        string                               `gorm:"not null;unique"`
	Permissions []permissionpostgres.PermissionModel `gorm:"many2many:role_permissions;joinForeignKey:RoleID;joinReferences:PermissionID;constraint:OnDelete:CASCADE;"`
}

func (RoleModel) TableName() string { return "roles" }

func roleModelFromDomain(role *roledomain.Role) *RoleModel {
	model := &RoleModel{
		BaseModel: postgres.BaseModelFromDomain(role.BaseModel),
		Name:      role.Name,
	}
	for _, permission := range role.Permissions {
		permissionModel := permissionpostgres.PermissionModel{
			BaseModel: postgres.BaseModelFromDomain(permission.BaseModel),
			Name:      permission.Name,
		}
		model.Permissions = append(model.Permissions, permissionModel)
	}
	return model
}

func roleDomainFromModel(m *RoleModel) *roledomain.Role {
	role := &roledomain.Role{
		BaseModel: m.BaseModel.ToDomain(),
		Name:      m.Name,
	}
	for _, permission := range m.Permissions {
		domainPermission := permissiondomain.Permission{
			BaseModel: permission.BaseModel.ToDomain(),
			Name:      permission.Name,
		}
		role.Permissions = append(role.Permissions, domainPermission)
	}
	return role
}
