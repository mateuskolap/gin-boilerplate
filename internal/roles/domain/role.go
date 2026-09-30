package domain

import (
	"context"
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	"gin-boilerplate/internal/domain/shared"
	permissiondomain "gin-boilerplate/internal/permissions/domain"

	"uuid"
)

const (
	RoleAdmin = "Admin"
	RoleUser  = "User"
)

type Role struct {
	shared.BaseModel
	Name string `json:"name" activity:"track"`

	Permissions []permissiondomain.Permission `json:"permissions,omitempty"`
}

func (r Role) ActivityLogSubjectType() activitylogdomain.ActivitySubjectType {
	return activitylogdomain.ActivitySubjectRole
}

func (r Role) ActivityLogID() uuid.UUID { return r.ID }

type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	GetByIDWithPermissions(ctx context.Context, id uuid.UUID) (*Role, error)
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[Role], error)

	// GetByName finds a role by its unique name.
	// Returns (nil, nil) if no role matches the name.
	GetByName(ctx context.Context, name string) (*Role, error)

	// AddPermissions adds permissions to a role.
	AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error

	// RemovePermissions removes permissions from a role.
	RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
}

type RoleUseCase interface {
	shared.BaseListUseCase[Role]

	shared.BaseFindUseCase[Role]

	shared.BaseDeleteUseCase

	// Create creates a new role.
	Create(ctx context.Context, role *Role) error

	// Update updates an existing role.
	Update(ctx context.Context, role *Role) error

	// AddPermissions adds permissions to a role.
	AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error

	// RemovePermissions removes permissions from a role.
	RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
}
