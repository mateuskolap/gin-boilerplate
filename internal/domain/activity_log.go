package domain

import (
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
	ID          uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Event       ActivityEvent       `json:"event" gorm:"size:64;not null"`
	ActorID     *uuid.UUID          `json:"actor_id,omitempty" gorm:"type:uuid"`
	SubjectType ActivitySubjectType `json:"subject_type" gorm:"size:32;not null"`
	SubjectID   uuid.UUID           `json:"subject_id" gorm:"type:uuid;not null"`
	Changes     map[string]any      `json:"changes" gorm:"serializer:json;type:jsonb;not null"`
	IPAddress   *string             `json:"ip_address,omitempty" gorm:"type:inet"`
	CreatedAt   time.Time           `json:"created_at"`
}

type ActivityLogRepository interface {
	shared.BaseRepository[ActivityLog]
}

type ActivityLogUseCase interface {
	shared.BaseListUseCase[ActivityLog]
}
