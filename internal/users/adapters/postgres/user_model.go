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
		ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
		Name: user.Name, Email: user.Email, Password: user.Password, AvatarKey: user.AvatarKey,
	}
	if user.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *user.DeletedAt, Valid: true}
	}
	for _, role := range user.Roles {
		model.Roles = append(model.Roles, rolespostgres.RoleModel{
			ID: role.ID, CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt,
			Name: role.Name,
		})
	}
	return model
}

func userDomainFromModel(model any) *userdomain.User {
	m := model.(*UserModel)
	user := &userdomain.User{
		Name: m.Name, Email: m.Email, Password: m.Password, AvatarKey: m.AvatarKey,
	}
	user.ID, user.CreatedAt, user.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	if m.DeletedAt.Valid {
		deletedAt := m.DeletedAt.Time
		user.DeletedAt = &deletedAt
	}
	for _, role := range m.Roles {
		user.Roles = append(user.Roles, roledomain.Role{
			ID: role.ID, CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt,
			Name: role.Name,
		})
	}
	return user
}
