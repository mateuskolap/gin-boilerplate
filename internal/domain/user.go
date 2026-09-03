package domain

import (
	"context"
	"uuid"
)

type User struct {
	BaseSoftDeleteModel
	Name     string `json:"name" gorm:"not null"`
	Email    string `json:"email" gorm:"not null;unique"`
	Password string `json:"-" gorm:"not null"`

	Roles []Role `json:"roles,omitempty" gorm:"many2many:user_roles;constraint:OnDelete:CASCADE;"`
}

type UserRepository interface {
	BaseRepository[User]

	// GetByEmail retrieves a user by their unique email address, optionally preloading relationships.
	// Returns (nil, nil) if no user matches the email.
	GetByEmail(ctx context.Context, email string, preloads ...string) (*User, error)

	// AddRoles associates roles with a user.
	AddRoles(ctx context.Context, user User, roleIDs []uuid.UUID) error

	// RemoveRoles disassociates roles from a user.
	RemoveRoles(ctx context.Context, user User, roleIDs []uuid.UUID) error
}

type UserUseCase interface {
	BaseListUseCase[User]

	BaseFindUseCase[User]

	// Register validates, hashes credentials, and creates a new user account.
	Register(ctx context.Context, user *User) error

	// Login verifies credentials and generates access and refresh tokens.
	Login(ctx context.Context, email string, password, ipAddress, userAgent string) (*AuthTokens, error)

	// Refresh verifies user has a valid refresh token and generates a signed JWT token string
	Refresh(ctx context.Context, refreshToken string, ipAddress, userAgent string) (*AuthTokens, error)

	// Logout invalidates a JWT token by adding its ID to the blacklist.
	Logout(ctx context.Context, accessToken, refreshToken string) error

	// UpdateProfile updates editable user profile fields.
	UpdateProfile(ctx context.Context, user *User) error

	// AddRoles associates roles with a user.
	AddRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error

	// RemoveRoles disassociates roles from a user.
	RemoveRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
}
