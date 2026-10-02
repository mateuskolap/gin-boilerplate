package postgres

import (
	"gin-boilerplate/internal/infra/postgres"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
)

type PermissionModel struct {
	postgres.BaseModel
	Name string `gorm:"unique;not null"`
}

func (PermissionModel) TableName() string { return "permissions" }

func permissionModelFromDomain(permission *permissiondomain.Permission) *PermissionModel {
	model := &PermissionModel{Name: permission.Name}
	model.BaseModel = postgres.BaseModelFromDomain(permission.BaseModel)
	return model
}

func permissionDomainFromModel(m *PermissionModel) *permissiondomain.Permission {
	permission := &permissiondomain.Permission{Name: m.Name}
	permission.BaseModel = m.BaseModel.ToDomain()
	return permission
}
