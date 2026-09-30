package application

import (
	"context"

	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	sharedapp "gin-boilerplate/internal/application"
	"gin-boilerplate/internal/domain/shared"

	"uuid"
)

var allowedActivityLogFilterFields = map[string]bool{
	"actor_id":     true,
	"created_at":   true,
	"event":        true,
	"subject_id":   true,
	"subject_type": true,
}

func NewActivityLogUseCase(repo activitylogdomain.ActivityLogRepository) activitylogdomain.ActivityLogUseCase {
	return sharedapp.NewBaseListUseCase(repo, allowedActivityLogFilterFields)
}

func newActivityLog(
	ctx context.Context,
	event activitylogdomain.ActivityEvent,
	subjectType activitylogdomain.ActivitySubjectType,
	subjectID uuid.UUID,
	changes map[string]any,
) *activitylogdomain.ActivityLog {
	return &activitylogdomain.ActivityLog{
		Event:       event,
		ActorID:     shared.ActorIDFromContext(ctx),
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Changes:     changes,
		IPAddress:   shared.RequestIPFromContext(ctx),
	}
}

func RecordActivity(
	ctx context.Context,
	repo activitylogdomain.ActivityLogRepository,
	event activitylogdomain.ActivityEvent,
	subjectType activitylogdomain.ActivitySubjectType,
	subjectID uuid.UUID,
	changes map[string]any,
) error {
	if err := repo.Create(ctx, newActivityLog(ctx, event, subjectType, subjectID, changes)); err != nil {
		return shared.NewAppError(shared.ErrTypeInternal, "Failed to record activity log", err)
	}
	return nil
}

func RelationChanges(name string, added, removed []uuid.UUID) map[string]any {
	return map[string]any{
		"relations": map[string]any{
			name: map[string]any{
				"added":   added,
				"removed": removed,
			},
		},
	}
}

func EffectiveRelationIDs(current, requested []uuid.UUID, adding bool) []uuid.UUID {
	existing := make(map[uuid.UUID]bool, len(current))
	for _, id := range current {
		existing[id] = true
	}

	result := make([]uuid.UUID, 0, len(requested))
	seen := make(map[uuid.UUID]bool, len(requested))
	for _, id := range requested {
		if seen[id] || existing[id] == adding {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}
