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

func activityLogModelFromDomain(entity *domain.ActivityLog) *ActivityLogModel {
	return &ActivityLogModel{
		ID: entity.ID, Event: string(entity.Event), ActorID: entity.ActorID,
		SubjectType: string(entity.SubjectType), SubjectID: entity.SubjectID,
		Changes: entity.Changes, IPAddress: entity.IPAddress, CreatedAt: entity.CreatedAt,
	}
}

func activityLogDomainFromModel(model *ActivityLogModel) *domain.ActivityLog {
	return &domain.ActivityLog{
		ID: model.ID, Event: domain.ActivityEvent(model.Event), ActorID: model.ActorID,
		SubjectType: domain.ActivitySubjectType(model.SubjectType), SubjectID: model.SubjectID,
		Changes: model.Changes, IPAddress: model.IPAddress, CreatedAt: model.CreatedAt,
	}
}
