package domain

import (
	"context"
)

type User struct {
	BaseSoftDeleteModel
	Name     string `json:"name" gorm:"not null"`
	Email    string `json:"email" gorm:"not null;unique"`
	Password string `json:"-" gorm:"not null"`
}

type UserRepository interface {
	BaseRepository[User]

	// GetByEmail retrieves a user by their unique email address.
	// Returns (nil, nil) if no user matches the email.
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type UserUseCase interface {
	BaseListUseCase[User]

	BaseFindUseCase[User]

	// Register validates, hashes credentials, and creates a new user account.
	Register(ctx context.Context, user *User) error

	// Login verifies credentials and generates a signed JWT token string.
	Login(ctx context.Context, email string, password string) (token string, err error)

	// Logout invalidates a JWT token by adding its ID to the blacklist.
	Logout(ctx context.Context, tokenString string) error

	// UpdateProfile updates editable user profile fields.
	UpdateProfile(ctx context.Context, user *User) error
}
