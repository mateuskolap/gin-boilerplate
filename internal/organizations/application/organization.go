package application

import (
	"context"
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	sharedapp "gin-boilerplate/internal/application"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	organizationdomain "gin-boilerplate/internal/organizations/domain"
)

var allowedOrganizationFilterFields = map[string]bool{
	"id": true, "created_at": true, "updated_at": true,
}

type organizationUseCase struct {
	shared.BaseListUseCase[organizationdomain.Organization]
	shared.BaseFindUseCase[organizationdomain.Organization]
	shared.BaseDeleteUseCase
	repository      organizationdomain.OrganizationRepository
	tx              port.TransactionManager
	activityLogRepo activitylogdomain.ActivityLogRepository
}

func NewOrganizationUseCase(
	repository organizationdomain.OrganizationRepository,
	tx port.TransactionManager,
	activityLogRepo activitylogdomain.ActivityLogRepository,
) organizationdomain.OrganizationUseCase {
	return &organizationUseCase{
		BaseListUseCase:   sharedapp.NewBaseListUseCase(repository, allowedOrganizationFilterFields),
		BaseFindUseCase:   sharedapp.NewBaseFindUseCase(repository),
		BaseDeleteUseCase: sharedapp.NewBaseDeleteUseCase(repository),
		repository:        repository,
	}
}

func (u *organizationUseCase) Create(ctx context.Context, entity *organizationdomain.Organization) error {
	return u.repository.Create(ctx, entity)
}

func (u *organizationUseCase) Update(ctx context.Context, entity *organizationdomain.Organization) error {
	return u.repository.Update(ctx, entity)
}
