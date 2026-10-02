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

func roleModelFromDomain(entity *roledomain.Role) *RoleModel {
	model := &RoleModel{
		BaseModel: postgres.BaseModelFromDomain(entity.BaseModel),
		Name:      entity.Name,
	}
	for _, permission := range entity.Permissions {
		permissionModel := permissionpostgres.PermissionModel{
			BaseModel: postgres.BaseModelFromDomain(permission.BaseModel),
			Name:      permission.Name,
		}
		model.Permissions = append(model.Permissions, permissionModel)
	}
	return model
}

func roleDomainFromModel(model *RoleModel) *roledomain.Role {
	entity := &roledomain.Role{
		BaseModel: model.BaseModel.ToDomain(),
		Name:      model.Name,
	}
	for _, permissionModel := range model.Permissions {
		permission := permissiondomain.Permission{
			BaseModel: permissionModel.BaseModel.ToDomain(),
			Name:      permissionModel.Name,
		}
		entity.Permissions = append(entity.Permissions, permission)
	}
	return entity
}
