package domain

import (
	"context"
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	"gin-boilerplate/internal/domain/shared"
	roledomain "gin-boilerplate/internal/roles/domain"
	"io"
	"uuid"
)

type User struct {
	shared.BaseSoftDeleteModel
	Name      string `json:"name" activity:"track"`
	Email     string `json:"email" activity:"track"`
	Password  string `json:"-" activity:"-"`
	AvatarKey string `json:"avatar_key,omitempty"`

	Roles []roledomain.Role `json:"roles,omitempty"`
}

func (u User) ActivityLogSubjectType() activitylogdomain.ActivitySubjectType {
	return activitylogdomain.ActivitySubjectUser
}

func (u User) ActivityLogID() uuid.UUID { return u.ID }

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByIDWithRoles(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[User], error)

	// GetByEmail retrieves a user by their unique email address.
	// Returns (nil, nil) if no user matches the email.
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByEmailWithRoles(ctx context.Context, email string) (*User, error)

	// AddRoles associates roles with a user.
	AddRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error

	// RemoveRoles disassociates roles from a user.
	RemoveRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
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
