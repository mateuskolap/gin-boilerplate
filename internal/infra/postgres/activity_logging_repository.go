package postgres

import (
	"context"
	"reflect"
	"strings"

	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	"gin-boilerplate/internal/domain/shared"

	"gorm.io/gorm"
	"uuid"
)

// activityLoggingRepository adds audit records around CRUD for explicitly
// auditable models while leaving the generic base repository unaware of logs.
type ActivityLoggingRepository[T any] struct {
	EntityRepository[T]
	db              *gorm.DB
	activityLogRepo activitylogdomain.ActivityLogRepository
}

type EntityRepository[T any] interface {
	Create(context.Context, *T) error
	GetByID(context.Context, uuid.UUID) (*T, error)
	FindOneBy(context.Context, string, []any, ...string) (*T, error)
	Update(context.Context, *T) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, shared.PaginationParams, []shared.Filter) (*shared.PaginatedResult[T], error)
}

func NewActivityLoggingRepository[T any](
	db *gorm.DB,
	base EntityRepository[T],
	activityLogRepo activitylogdomain.ActivityLogRepository,
) *ActivityLoggingRepository[T] {
	return &ActivityLoggingRepository[T]{
		EntityRepository: base,
		db:               db,
		activityLogRepo:  activityLogRepo,
	}
}

func (r *ActivityLoggingRepository[T]) DB(ctx context.Context) *gorm.DB {
	return GetTxFromContext(ctx, r.db)
}

func (r *ActivityLoggingRepository[T]) Create(ctx context.Context, entity *T) error {
	if !shouldAuditModel(entity) {
		return r.EntityRepository.Create(ctx, entity)
	}
	return r.withTransaction(ctx, func(txCtx context.Context) error {
		if err := r.EntityRepository.Create(txCtx, entity); err != nil {
			return err
		}
		return createModelActivity(txCtx, r.activityLogRepo, entity, "created", nil)
	})
}

func (r *ActivityLoggingRepository[T]) Update(ctx context.Context, entity *T) error {
	if !shouldAuditModel(entity) {
		return r.EntityRepository.Update(ctx, entity)
	}
	return r.withTransaction(ctx, func(txCtx context.Context) error {
		oldEntity, err := r.EntityRepository.GetByID(txCtx, activityLogID(entity))
		if err != nil {
			return err
		}
		if err := r.EntityRepository.Update(txCtx, entity); err != nil {
			return err
		}
		if oldEntity == nil {
			return nil
		}

		oldAttributes := activityAttributes(oldEntity)
		return createModelActivity(txCtx, r.activityLogRepo, entity, "updated", oldAttributes)
	})
}

func (r *ActivityLoggingRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	var entity T
	if !shouldAuditModel(&entity) {
		return r.EntityRepository.Delete(ctx, id)
	}
	return r.withTransaction(ctx, func(txCtx context.Context) error {
		oldEntity, err := r.EntityRepository.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if err := r.EntityRepository.Delete(txCtx, id); err != nil {
			return err
		}
		if oldEntity == nil {
			return nil
		}
		return createModelActivity(txCtx, r.activityLogRepo, oldEntity, "deleted", activityAttributes(oldEntity))
	})
}

func (r *ActivityLoggingRepository[T]) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return fn(ctx)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func shouldAuditModel(entity any) bool {
	_, ok := entity.(activitylogdomain.ActivityLoggable)
	return ok && len(activityAttributes(entity)) > 0
}

func activityLogID(entity any) uuid.UUID {
	return entity.(activitylogdomain.ActivityLoggable).ActivityLogID()
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
	repo activitylogdomain.ActivityLogRepository,
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

func newModelActivity(ctx context.Context, entity any, action string, oldAttributes map[string]any) *activitylogdomain.ActivityLog {
	model, ok := entity.(activitylogdomain.ActivityLoggable)
	actorID := shared.ActorIDFromContext(ctx)
	attributes := activityAttributes(entity)
	if !ok {
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

	return &activitylogdomain.ActivityLog{
		Event:       activitylogdomain.ActivityEvent(string(model.ActivityLogSubjectType()) + "." + action),
		ActorID:     actorID,
		SubjectType: model.ActivityLogSubjectType(),
		SubjectID:   model.ActivityLogID(),
		Changes: map[string]any{
			"attributes": map[string]any{
				"old": oldAttributes,
				"new": newAttributes,
			},
		},
		IPAddress: shared.RequestIPFromContext(ctx),
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
