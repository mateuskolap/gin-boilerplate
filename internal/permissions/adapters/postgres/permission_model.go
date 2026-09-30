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
	return &PermissionModel{
		ID: permission.ID, CreatedAt: permission.CreatedAt, UpdatedAt: permission.UpdatedAt,
		Name: permission.Name,
	}
}

func permissionDomainFromModel(model any) *permissiondomain.Permission {
	m := model.(*PermissionModel)
	return &permissiondomain.Permission{
		ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		Name: m.Name,
	}
}
