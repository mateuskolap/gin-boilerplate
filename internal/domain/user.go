package domain

import (
	"context"

	"github.com/google/uuid"
)

type User struct {
	BaseSoftDeleteModel
	Name     string `json:"name" gorm:"not null"`
	Email    string `json:"email" gorm:"not null;unique"`
	Password string `json:"-" gorm:"not null"`
}

type UserRepository interface {
	BaseRepository[User]
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type UserUseCase interface {
	Register(ctx context.Context, user *User) error
	Login(ctx context.Context, email string, password string) (token string, err error)
	Logout(ctx context.Context, tokenString string) error
	GetProfile(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateProfile(ctx context.Context, user *User) error
}
