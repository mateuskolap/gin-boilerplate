package postgres

import (
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	organizationdomain "gin-boilerplate/internal/organizations/domain"

	"gorm.io/gorm"
)

type organizationRepository struct {
	*postgresinfra.ActivityLoggingRepository[organizationdomain.Organization]
}

var _ organizationdomain.OrganizationRepository = (*organizationRepository)(nil)

func NewOrganizationRepository(db *gorm.DB, activityLogRepo activitylogdomain.ActivityLogRepository) organizationdomain.OrganizationRepository {
	return &organizationRepository{
		ActivityLoggingRepository: postgresinfra.NewActivityLoggingRepository(
			db,
			postgresinfra.NewBaseRepository(db,
				func() *OrganizationModel { return &OrganizationModel{} },
				func(entity *organizationdomain.Organization) *OrganizationModel {
					return organizationModelFromDomain(entity)
				},
				organizationDomainFromModel,
			),
			activityLogRepo,
		),
	}
}
