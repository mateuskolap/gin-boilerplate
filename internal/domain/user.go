package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"uuid"
)

type User struct {
	shared.BaseSoftDeleteModel
	Name     string `json:"name" gorm:"not null"`
	Email    string `json:"email" gorm:"not null;unique"`
	Password string `json:"-" gorm:"not null"`

	Roles []Role `json:"roles,omitempty" gorm:"many2many:user_roles;constraint:OnDelete:CASCADE;"`
}

type UserRepository interface {
	shared.BaseRepository[User]

	// GetByEmail retrieves a user by their unique email address, optionally preloading relationships.
	// Returns (nil, nil) if no user matches the email.
	GetByEmail(ctx context.Context, email string, preloads ...string) (*User, error)

	// AddRoles associates roles with a user.
	AddRoles(ctx context.Context, user User, roleIDs []uuid.UUID) error

	// RemoveRoles disassociates roles from a user.
	RemoveRoles(ctx context.Context, user User, roleIDs []uuid.UUID) error
}

type UserUseCase interface {
	shared.BaseListUseCase[User]

	shared.BaseFindUseCase[User]

	shared.BaseDeleteUseCase

	// UpdateProfile updates editable user profile fields.
	UpdateProfile(ctx context.Context, user *User) error

	// AddRoles associates roles with a user.
	AddRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error

	// RemoveRoles disassociates roles from a user.
	RemoveRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
}
