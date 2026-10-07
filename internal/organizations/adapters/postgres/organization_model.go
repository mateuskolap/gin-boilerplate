package postgres

import (
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	organizationdomain "gin-boilerplate/internal/organizations/domain"

	"gorm.io/gorm"
)

type OrganizationModel struct {
	postgresinfra.BaseModel
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_organizations_name_active,where:deleted_at IS NULL"`
}

func (OrganizationModel) TableName() string { return "organizations" }

func organizationModelFromDomain(entity *organizationdomain.Organization) *OrganizationModel {
	model := &OrganizationModel{
		BaseModel: postgresinfra.BaseModelFromDomain(entity.BaseModel),
		Name:      entity.Name,
	}
	if entity.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
	return model
}

func organizationDomainFromModel(model *OrganizationModel) *organizationdomain.Organization {
	entity := &organizationdomain.Organization{
		BaseModel: model.BaseModel.ToDomain(),
		Name:      model.Name,
	}

	if model.DeletedAt.Valid {
		deletedAt := model.DeletedAt.Time
		entity.DeletedAt = &deletedAt
	}
	return entity
}
