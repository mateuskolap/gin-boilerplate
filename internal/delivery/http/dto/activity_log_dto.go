package dto

import (
	"time"

	"gin-boilerplate/internal/domain"

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

type PaginatedActivityLogResponse = PaginatedResponse[ActivityLogResponse]

func ToActivityLogResponse(activity *domain.ActivityLog) ActivityLogResponse {
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
