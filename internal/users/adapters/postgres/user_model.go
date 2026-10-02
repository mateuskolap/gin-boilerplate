package postgres

import (
	"gin-boilerplate/internal/infra/postgres"
	rolespostgres "gin-boilerplate/internal/roles/adapters/postgres"
	roledomain "gin-boilerplate/internal/roles/domain"
	userdomain "gin-boilerplate/internal/users/domain"

	"gorm.io/gorm"
)

type UserModel struct {
	postgres.BaseModel
	Name      string `gorm:"not null"`
	Email     string `gorm:"not null;uniqueIndex:idx_users_email_active,expression:LOWER(email),where:deleted_at IS NULL"`
	Password  string `gorm:"not null"`
	AvatarKey string
	DeletedAt gorm.DeletedAt            `gorm:"index"`
	Roles     []rolespostgres.RoleModel `gorm:"many2many:user_roles;joinForeignKey:UserID;joinReferences:RoleID;constraint:OnDelete:CASCADE;"`
}

func (UserModel) TableName() string { return "users" }

func userModelFromDomain(user *userdomain.User) *UserModel {
	model := &UserModel{
		BaseModel: postgres.BaseModelFromDomain(user.BaseSoftDeleteModel.BaseModel),
		Name:      user.Name, Email: user.Email, Password: user.Password, AvatarKey: user.AvatarKey,
	}
	if user.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *user.DeletedAt, Valid: true}
	}
	for _, role := range user.Roles {
		roleModel := rolespostgres.RoleModel{
			BaseModel: postgres.BaseModelFromDomain(role.BaseModel),
			Name:      role.Name,
		}
		model.Roles = append(model.Roles, roleModel)
	}
	return model
}

func userDomainFromModel(m *UserModel) *userdomain.User {
	user := &userdomain.User{
		BaseModel: m.BaseModel.ToDomain(),
		Name:      m.Name, Email: m.Email, Password: m.Password, AvatarKey: m.AvatarKey,
	}
	if m.DeletedAt.Valid {
		deletedAt := m.DeletedAt.Time
		user.DeletedAt = &deletedAt
	}
	for _, role := range m.Roles {
		domainRole := roledomain.Role{
			BaseModel: role.BaseModel.ToDomain(),
			Name:      role.Name,
		}
		user.Roles = append(user.Roles, domainRole)
	}
	return user
}
