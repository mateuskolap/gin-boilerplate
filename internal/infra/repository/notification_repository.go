package repository

import (
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
)

type notificationRepository struct {
	*baseRepository[domain.Notification]
}

func NewNotificationRepository(db *gorm.DB) domain.NotificationRepository {
	return &notificationRepository{
		baseRepository: newBaseRepository[domain.Notification](db),
	}
}
