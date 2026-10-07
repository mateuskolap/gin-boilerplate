package domain

import (
	"context"
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	"gin-boilerplate/internal/domain/shared"
	"uuid"
)

type Organization struct {
	shared.BaseSoftDeleteModel
	Name string `json:"name" activity:"track"`
}

var _ activitylogdomain.ActivityLoggable = Organization{}

func (o Organization) ActivityLogSubjectType() activitylogdomain.ActivitySubjectType {
	return activitylogdomain.ActivitySubjectOrganization
}

func (o Organization) ActivityLogID() uuid.UUID { return o.ID }

type OrganizationRepository interface {
	Create(context.Context, *Organization) error
	GetByID(context.Context, uuid.UUID) (*Organization, error)
	Update(context.Context, *Organization) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, shared.PaginationParams, []shared.Filter) (*shared.PaginatedResult[Organization], error)
}

type OrganizationUseCase interface {
	shared.BaseListUseCase[Organization]
	shared.BaseFindUseCase[Organization]
	shared.BaseDeleteUseCase
	Create(context.Context, *Organization) error
	Update(context.Context, *Organization) error
}
