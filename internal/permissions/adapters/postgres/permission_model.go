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

func permissionModelFromDomain(entity *permissiondomain.Permission) *PermissionModel {
	model := &PermissionModel{Name: entity.Name}
	model.BaseModel = postgres.BaseModelFromDomain(entity.BaseModel)
	return model
}

func permissionDomainFromModel(model *PermissionModel) *permissiondomain.Permission {
	entity := &permissiondomain.Permission{Name: model.Name}
	entity.BaseModel = model.BaseModel.ToDomain()
	return entity
}
