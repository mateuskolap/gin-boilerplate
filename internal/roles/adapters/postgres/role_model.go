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
	model := &RoleModel{ID: role.ID, CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt, Name: role.Name}
	for _, permission := range role.Permissions {
		model.Permissions = append(model.Permissions, permissionpostgres.PermissionModel{
			ID: permission.ID, CreatedAt: permission.CreatedAt, UpdatedAt: permission.UpdatedAt,
			Name: permission.Name,
		})
	}
	return model
}

func roleDomainFromModel(model any) *roledomain.Role {
	m := model.(*RoleModel)
	role := &roledomain.Role{
		ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		Name: m.Name,
	}
	for _, permission := range m.Permissions {
		role.Permissions = append(role.Permissions, permissiondomain.Permission{
			ID: permission.ID, CreatedAt: permission.CreatedAt, UpdatedAt: permission.UpdatedAt,
			Name: permission.Name,
		})
	}
	return role
}
