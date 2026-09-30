package postgres

import (
	"context"
	"errors"
	"fmt"
	"gin-boilerplate/config"
	rolespostgres "gin-boilerplate/internal/roles/adapters/postgres"
	roledomain "gin-boilerplate/internal/roles/domain"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type userSeeder struct {
	cfg *config.Config
}

func NewUserSeeder(cfg *config.Config) interface {
	Seed(context.Context, *gorm.DB) error
} {
	return &userSeeder{cfg: cfg}
}

func (s *userSeeder) Seed(ctx context.Context, tx *gorm.DB) error {
	adminEmail := s.cfg.AdminEmail
	adminPassword := s.cfg.AdminPassword
	adminName := s.cfg.AdminName

	var adminRole rolespostgres.RoleModel
	if err := tx.WithContext(ctx).Where("name = ?", roledomain.RoleAdmin).First(&adminRole).Error; err != nil {
		return fmt.Errorf("admin role not found: %w", err)
	}

	var existingUser UserModel
	err := tx.WithContext(ctx).Where("email = ?", adminEmail).First(&existingUser).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing admin user: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}

		newUser := UserModel{
			Name:     adminName,
			Email:    adminEmail,
			Password: string(hashedPassword),
		}

		if err := tx.WithContext(ctx).Create(&newUser).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		if err := tx.WithContext(ctx).Model(&newUser).Association("Roles").Append(&adminRole); err != nil {
			return fmt.Errorf("failed to assign admin role: %w", err)
		}
	} else {
		if err := tx.WithContext(ctx).
			Model(&existingUser).
			Association("Roles").
			Append(&adminRole); err != nil {
			return fmt.Errorf("failed to link admin role to existing user: %w", err)
		}
	}

	return nil
}
