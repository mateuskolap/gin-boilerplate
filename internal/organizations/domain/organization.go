package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"uuid"
)

type Organization struct {
	shared.BaseSoftDeleteModel
	Name string `json:"name"`
}

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
