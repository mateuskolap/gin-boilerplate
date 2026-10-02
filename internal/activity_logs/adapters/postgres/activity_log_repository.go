package postgres

import (
	"gin-boilerplate/internal/activity_logs/domain"
	postgresinfra "gin-boilerplate/internal/infra/postgres"

	"gorm.io/gorm"
)

func NewActivityLogRepository(db *gorm.DB) domain.ActivityLogRepository {
	return postgresinfra.NewBaseRepository(db,
		func() *ActivityLogModel { return &ActivityLogModel{} },
		func(activity *domain.ActivityLog) *ActivityLogModel { return activityLogModelFromDomain(activity) },
		activityLogDomainFromModel,
	)
}
