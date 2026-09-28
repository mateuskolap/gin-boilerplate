package repository

import (
	"context"
	"reflect"
	"strings"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"

	"gorm.io/gorm"
	"uuid"
)

// activityLoggingRepository adds audit records around CRUD for explicitly
// auditable models while leaving the generic base repository unaware of logs.
type activityLoggingRepository[T any] struct {
	shared.BaseRepository[T]
	db              *gorm.DB
	activityLogRepo domain.ActivityLogRepository
}

func newActivityLoggingRepository[T any](
	db *gorm.DB,
	base shared.BaseRepository[T],
	activityLogRepo domain.ActivityLogRepository,
) *activityLoggingRepository[T] {
	return &activityLoggingRepository[T]{
		BaseRepository:  base,
		db:              db,
		activityLogRepo: activityLogRepo,
	}
}

func (r *activityLoggingRepository[T]) getDB(ctx context.Context) *gorm.DB {
	return GetTxFromContext(ctx, r.db)
}

func (r *activityLoggingRepository[T]) Create(ctx context.Context, entity *T) error {
	if !shouldAuditModel(ctx, entity) {
		return r.BaseRepository.Create(ctx, entity)
	}
	return r.withTransaction(ctx, func(txCtx context.Context) error {
		if err := r.BaseRepository.Create(txCtx, entity); err != nil {
			return err
		}
		return createModelActivity(txCtx, r.activityLogRepo, entity, "created", nil)
	})
}

func (r *activityLoggingRepository[T]) Update(ctx context.Context, entity *T) error {
	if !shouldAuditModel(ctx, entity) {
		return r.BaseRepository.Update(ctx, entity)
	}
	return r.withTransaction(ctx, func(txCtx context.Context) error {
		oldEntity, err := r.BaseRepository.GetByID(txCtx, activityLogID(entity))
		if err != nil {
			return err
		}
		if err := r.BaseRepository.Update(txCtx, entity); err != nil {
			return err
		}
		if oldEntity == nil {
			return nil
		}

		oldAttributes := activityAttributes(oldEntity)
		return createModelActivity(txCtx, r.activityLogRepo, entity, "updated", oldAttributes)
	})
}

func (r *activityLoggingRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	var entity T
	if !shouldAuditModel(ctx, &entity) {
		return r.BaseRepository.Delete(ctx, id)
	}
	return r.withTransaction(ctx, func(txCtx context.Context) error {
		oldEntity, err := r.BaseRepository.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if err := r.BaseRepository.Delete(txCtx, id); err != nil {
			return err
		}
		if oldEntity == nil {
			return nil
		}
		return createModelActivity(txCtx, r.activityLogRepo, oldEntity, "deleted", activityAttributes(oldEntity))
	})
}

func (r *activityLoggingRepository[T]) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return fn(ctx)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func shouldAuditModel(ctx context.Context, entity any) bool {
	_, ok := entity.(domain.ActivityLoggable)
	return ok && shared.ActorIDFromContext(ctx) != nil && len(activityAttributes(entity)) > 0
}

func activityLogID(entity any) uuid.UUID {
	return entity.(domain.ActivityLoggable).ActivityLogID()
}

func activityAttributes(entity any) map[string]any {
	value := reflect.Indirect(reflect.ValueOf(entity))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil
	}

	attributes := make(map[string]any)
	typeOfValue := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typeOfValue.Field(i)
		if field.Tag.Get("activity") != "track" {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" || !value.Field(i).CanInterface() {
			continue
		}
		attributes[name] = value.Field(i).Interface()
	}
	return attributes
}

func createModelActivity(
	ctx context.Context,
	repo domain.ActivityLogRepository,
	entity any,
	action string,
	oldAttributes map[string]any,
) error {
	activity := newModelActivity(ctx, entity, action, oldAttributes)
	if activity == nil {
		return nil
	}
	return repo.Create(ctx, activity)
}

func newModelActivity(ctx context.Context, entity any, action string, oldAttributes map[string]any) *domain.ActivityLog {
	model, ok := entity.(domain.ActivityLoggable)
	actorID := shared.ActorIDFromContext(ctx)
	attributes := activityAttributes(entity)
	if !ok || actorID == nil {
		return nil
	}
	if action == "updated" {
		oldAttributes, attributes = changedActivityAttributes(oldAttributes, attributes)
		if len(oldAttributes) == 0 && len(attributes) == 0 {
			return nil
		}
	} else if len(attributes) == 0 {
		return nil
	}

	newAttributes := attributes
	if action == "deleted" {
		newAttributes = map[string]any{}
	}
	if oldAttributes == nil {
		oldAttributes = map[string]any{}
	}

	return &domain.ActivityLog{
		Event:       domain.ActivityEvent(string(model.ActivityLogSubjectType()) + "." + action),
		ActorID:     actorID,
		SubjectType: model.ActivityLogSubjectType(),
		SubjectID:   model.ActivityLogID(),
		Changes: map[string]any{
			"attributes": map[string]any{
				"old": oldAttributes,
				"new": newAttributes,
			},
		},
		RequestID: shared.RequestIDFromContext(ctx),
	}
}

func changedActivityAttributes(old, current map[string]any) (map[string]any, map[string]any) {
	oldChanges := make(map[string]any)
	newChanges := make(map[string]any)
	for name, value := range current {
		oldValue, exists := old[name]
		if exists && reflect.DeepEqual(oldValue, value) {
			continue
		}
		if exists {
			oldChanges[name] = oldValue
		}
		newChanges[name] = value
	}
	for name, value := range old {
		if _, exists := current[name]; !exists {
			oldChanges[name] = value
		}
	}
	return oldChanges, newChanges
}
