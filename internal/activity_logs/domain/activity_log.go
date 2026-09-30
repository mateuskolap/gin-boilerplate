package domain

import (
	"context"
	"time"

	"gin-boilerplate/internal/domain/shared"

	"uuid"
)

type ActivityEvent string

const (
	ActivityUserUpdated            ActivityEvent = "user.updated"
	ActivityUserDeleted            ActivityEvent = "user.deleted"
	ActivityUserRolesAdded         ActivityEvent = "user.roles_added"
	ActivityUserRolesRemoved       ActivityEvent = "user.roles_removed"
	ActivityRoleCreated            ActivityEvent = "role.created"
	ActivityRoleUpdated            ActivityEvent = "role.updated"
	ActivityRoleDeleted            ActivityEvent = "role.deleted"
	ActivityRolePermissionsAdded   ActivityEvent = "role.permissions_added"
	ActivityRolePermissionsRemoved ActivityEvent = "role.permissions_removed"
)

type ActivitySubjectType string

const (
	ActivitySubjectUser ActivitySubjectType = "user"
	ActivitySubjectRole ActivitySubjectType = "role"
)

// ActivityLoggable marks an entity whose CRUD changes are automatically audited.
type ActivityLoggable interface {
	ActivityLogSubjectType() ActivitySubjectType
	ActivityLogID() uuid.UUID
}

// ActivityLog is an immutable audit record for an administrative change.
type ActivityLog struct {
	ID          uuid.UUID           `json:"id"`
	Event       ActivityEvent       `json:"event"`
	ActorID     *uuid.UUID          `json:"actor_id,omitempty"`
	SubjectType ActivitySubjectType `json:"subject_type"`
	SubjectID   uuid.UUID           `json:"subject_id"`
	Changes     map[string]any      `json:"changes"`
	IPAddress   *string             `json:"ip_address,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
}

type ActivityLogRepository interface {
	Create(ctx context.Context, activity *ActivityLog) error
	List(ctx context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[ActivityLog], error)
}

type ActivityLogUseCase interface {
	shared.BaseListUseCase[ActivityLog]
}
