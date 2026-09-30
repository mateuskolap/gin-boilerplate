package postgres

import (
	"gin-boilerplate/internal/activity_logs/domain"
	postgresinfra "gin-boilerplate/internal/infra/postgres"

	"gorm.io/gorm"
)

func NewActivityLogRepository(db *gorm.DB) domain.ActivityLogRepository {
	return postgresinfra.NewBaseRepository(db,
		func() any { return &ActivityLogModel{} },
		func(activity *domain.ActivityLog) any { return activityLogModelFromDomain(activity) },
		activityLogDomainFromModel,
	)
}
