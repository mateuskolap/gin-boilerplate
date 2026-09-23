package repository

import (
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
)

func NewPermissionRepository(db *gorm.DB) domain.PermissionRepository {
	return newBaseRepository[domain.Permission](db)
}
