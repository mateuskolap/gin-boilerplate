package dto

import (
	"time"

	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	"gin-boilerplate/internal/delivery/http/dto"

	"uuid"
)

type ActivityLogResponse struct {
	ID          uuid.UUID      `json:"id"`
	Event       string         `json:"event"`
	ActorID     *uuid.UUID     `json:"actor_id"`
	SubjectType string         `json:"subject_type"`
	SubjectID   uuid.UUID      `json:"subject_id"`
	Changes     map[string]any `json:"changes"`
	IPAddress   *string        `json:"ip_address,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type PaginatedActivityLogResponse = dto.PaginatedResponse[ActivityLogResponse]

func ToActivityLogResponse(activity *activitylogdomain.ActivityLog) ActivityLogResponse {
	return ActivityLogResponse{
		ID:          activity.ID,
		Event:       string(activity.Event),
		ActorID:     activity.ActorID,
		SubjectType: string(activity.SubjectType),
		SubjectID:   activity.SubjectID,
		Changes:     activity.Changes,
		IPAddress:   activity.IPAddress,
		CreatedAt:   activity.CreatedAt,
	}
}
