package repository

import (
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
)

func NewActivityLogRepository(db *gorm.DB) domain.ActivityLogRepository {
	return newBaseRepository[domain.ActivityLog](db)
}
