package postgres

import (
	"gin-boilerplate/internal/infra/postgres"
	organizationpostgres "gin-boilerplate/internal/organizations/adapters/postgres"
	organizationdomain "gin-boilerplate/internal/organizations/domain"
	rolespostgres "gin-boilerplate/internal/roles/adapters/postgres"
	roledomain "gin-boilerplate/internal/roles/domain"
	userdomain "gin-boilerplate/internal/users/domain"
	"uuid"

	"gorm.io/gorm"
)

type UserModel struct {
	postgres.BaseModel
	Name           string `gorm:"not null"`
	Email          string `gorm:"not null;uniqueIndex:idx_users_email_active,expression:LOWER(email),where:deleted_at IS NULL"`
	Password       string `gorm:"not null"`
	AvatarKey      string
	OrganizationID *uuid.UUID     `gorm:"type:uuid;index"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	Organization *organizationpostgres.OrganizationModel `gorm:"foreignKey:OrganizationID"`
	Roles        []rolespostgres.RoleModel               `gorm:"many2many:user_roles;joinForeignKey:UserID;joinReferences:RoleID;constraint:OnDelete:CASCADE;"`
}

func (UserModel) TableName() string { return "users" }

func userModelFromDomain(entity *userdomain.User) *UserModel {
	model := &UserModel{
		BaseModel: postgres.BaseModelFromDomain(entity.BaseSoftDeleteModel.BaseModel),
		Name:      entity.Name, Email: entity.Email, Password: entity.Password, AvatarKey: entity.AvatarKey,
	}
	if entity.OrganizationID != uuid.Nil() {
		organizationID := entity.OrganizationID
		model.OrganizationID = &organizationID
	}
	if entity.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
	for _, role := range entity.Roles {
		roleModel := rolespostgres.RoleModel{
			BaseModel: postgres.BaseModelFromDomain(role.BaseModel),
			Name:      role.Name,
		}
		model.Roles = append(model.Roles, roleModel)
	}
	return model
}

func userDomainFromModel(model *UserModel) *userdomain.User {
	entity := &userdomain.User{
		BaseModel: model.BaseModel.ToDomain(),
		Name:      model.Name, Email: model.Email, Password: model.Password, AvatarKey: model.AvatarKey,
	}
	if model.OrganizationID != nil {
		entity.OrganizationID = *model.OrganizationID
	}
	if model.Organization != nil {
		organization := &organizationdomain.Organization{
			BaseModel: model.Organization.BaseModel.ToDomain(),
			Name:      model.Organization.Name,
		}
		if model.Organization.DeletedAt.Valid {
			deletedAt := model.Organization.DeletedAt.Time
			organization.DeletedAt = &deletedAt
		}
		entity.Organization = organization
	}
	if model.DeletedAt.Valid {
		deletedAt := model.DeletedAt.Time
		entity.DeletedAt = &deletedAt
	}
	for _, roleModel := range model.Roles {
		role := roledomain.Role{
			BaseModel: roleModel.BaseModel.ToDomain(),
			Name:      roleModel.Name,
		}
		entity.Roles = append(entity.Roles, role)
	}
	return entity
}
