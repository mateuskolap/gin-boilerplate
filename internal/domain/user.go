package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"io"
	"uuid"
)

type User struct {
	shared.BaseSoftDeleteModel
	Name      string `json:"name" gorm:"not null"`
	Email     string `json:"email" gorm:"not null;uniqueIndex:idx_users_email_active,where:deleted_at IS NULL"`
	Password  string `json:"-" gorm:"not null"`
	AvatarKey string `json:"avatar_key,omitempty"`

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

	// UpdateImage updates the user's profile image.
	UpdateImage(ctx context.Context, userID uuid.UUID, file io.Reader) error

	// RemoveImage removes the user's profile image.
	RemoveImage(ctx context.Context, userID uuid.UUID) error

	// GetImage opens the user's profile image. The caller must close its content.
	GetImage(ctx context.Context, userID uuid.UUID) (shared.ImageStream, error)
}
