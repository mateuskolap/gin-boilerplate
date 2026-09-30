package postgres

import (
	"time"

	"gin-boilerplate/internal/activity_logs/domain"
	"uuid"
)

type ActivityLogModel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Event       string         `gorm:"size:64;not null"`
	ActorID     *uuid.UUID     `gorm:"type:uuid"`
	SubjectType string         `gorm:"size:32;not null"`
	SubjectID   uuid.UUID      `gorm:"type:uuid;not null"`
	Changes     map[string]any `gorm:"serializer:json;type:jsonb;not null"`
	IPAddress   *string        `gorm:"type:inet"`
	CreatedAt   time.Time
}

func (ActivityLogModel) TableName() string { return "activity_logs" }

func activityLogModelFromDomain(activity *domain.ActivityLog) *ActivityLogModel {
	return &ActivityLogModel{
		ID: activity.ID, Event: string(activity.Event), ActorID: activity.ActorID,
		SubjectType: string(activity.SubjectType), SubjectID: activity.SubjectID,
		Changes: activity.Changes, IPAddress: activity.IPAddress, CreatedAt: activity.CreatedAt,
	}
}

func activityLogDomainFromModel(model any) *domain.ActivityLog {
	m := model.(*ActivityLogModel)
	return &domain.ActivityLog{
		ID: m.ID, Event: domain.ActivityEvent(m.Event), ActorID: m.ActorID,
		SubjectType: domain.ActivitySubjectType(m.SubjectType), SubjectID: m.SubjectID,
		Changes: m.Changes, IPAddress: m.IPAddress, CreatedAt: m.CreatedAt,
	}
}
