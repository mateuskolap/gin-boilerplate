package application

import (
	"context"
	"errors"
	activitylogapp "gin-boilerplate/internal/activity_logs/application"
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	sharedapp "gin-boilerplate/internal/application"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	roledomain "gin-boilerplate/internal/roles/domain"
	"strings"
	"unicode/utf8"

	"uuid"
)

var allowedRoleFilterFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
	"id":         true,
}

type roleUseCase struct {
	shared.BaseListUseCase[roledomain.Role]
	shared.BaseDeleteUseCase
	roleRepo        roledomain.RoleRepository
	tx              port.TransactionManager
	activityLogRepo activitylogdomain.ActivityLogRepository
}

func NewRoleUseCase(
	roleRepo roledomain.RoleRepository,
	tx port.TransactionManager,
	activityLogRepo activitylogdomain.ActivityLogRepository,
) roledomain.RoleUseCase {
	return &roleUseCase{
		BaseListUseCase: sharedapp.NewBaseListUseCase(
			roleRepo,
			allowedRoleFilterFields,
		),
		BaseDeleteUseCase: sharedapp.NewBaseDeleteUseCase(roleRepo),
		roleRepo:          roleRepo,
		tx:                tx,
		activityLogRepo:   activityLogRepo,
	}
}

func (r *roleUseCase) Find(ctx context.Context, id uuid.UUID) (*roledomain.Role, error) {
	return sharedapp.FindByIDUsing(ctx, id, r.roleRepo.GetByIDWithPermissions)
}

func (r *roleUseCase) Create(ctx context.Context, role *roledomain.Role) error {
	if err := normalizeRoleName(role); err != nil {
		return err
	}
	existingRole, err := r.roleRepo.GetByName(ctx, role.Name)
	if err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"There was a problem verifying the role name",
			err,
		)
	}

	if existingRole != nil {
		return shared.NewAppError(
			shared.ErrTypeConflict,
			"This role already exists",
			nil,
		)
	}

	if err := r.roleRepo.Create(ctx, role); err != nil {
		if errors.Is(err, shared.ErrConflict) {
			return shared.NewAppError(shared.ErrTypeConflict, "This role already exists", err)
		}
		return shared.NewAppError(shared.ErrTypeInternal, "Failed to create role", err)
	}

	return nil
}

func (r *roleUseCase) Update(ctx context.Context, role *roledomain.Role) error {
	if err := normalizeRoleName(role); err != nil {
		return err
	}
	existingRole, err := sharedapp.FindByID(ctx, r.roleRepo, role.ID)
	if err != nil {
		return err
	}

	if existingRole.Name != role.Name {
		existingRole.Name = role.Name
		if err = r.roleRepo.Update(ctx, existingRole); err != nil {
			if errors.Is(err, shared.ErrConflict) {
				return shared.NewAppError(shared.ErrTypeConflict, "This role already exists", err)
			}
			return shared.NewAppError(shared.ErrTypeInternal, "Failed to update role", err)
		}
	}

	*role = *existingRole
	return nil
}

func normalizeRoleName(role *roledomain.Role) error {
	role.Name = strings.TrimSpace(role.Name)
	if nameLength := utf8.RuneCountInString(role.Name); nameLength < 2 || nameLength > 100 {
		return shared.NewAppError(
			shared.ErrTypeValidation,
			"Role name must contain between 2 and 100 characters",
			nil,
		)
	}
	return nil
}

func (r *roleUseCase) AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	role, err := sharedapp.FindByIDUsing(ctx, roleID, r.roleRepo.GetByIDWithPermissions)
	if err != nil {
		return err
	}

	currentPermissionIDs := make([]uuid.UUID, len(role.Permissions))
	for i, permission := range role.Permissions {
		currentPermissionIDs[i] = permission.ID
	}
	addedPermissionIDs := activitylogapp.EffectiveRelationIDs(currentPermissionIDs, permissionIDs, true)
	if len(addedPermissionIDs) == 0 {
		return nil
	}
	return r.tx.Do(ctx, func(txCtx context.Context) error {
		if err := r.roleRepo.AddPermissions(txCtx, role.ID, addedPermissionIDs); err != nil {
			return shared.NewAppError(shared.ErrTypeInternal, "Failed to add permissions to role", err)
		}
		return activitylogapp.RecordActivity(txCtx, r.activityLogRepo, activitylogdomain.ActivityRolePermissionsAdded, activitylogdomain.ActivitySubjectRole, role.ID,
			activitylogapp.RelationChanges("permissions", addedPermissionIDs, []uuid.UUID{}))
	})
}

func (r *roleUseCase) RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	role, err := sharedapp.FindByIDUsing(ctx, roleID, r.roleRepo.GetByIDWithPermissions)
	if err != nil {
		return err
	}

	currentPermissionIDs := make([]uuid.UUID, len(role.Permissions))
	for i, permission := range role.Permissions {
		currentPermissionIDs[i] = permission.ID
	}
	removedPermissionIDs := activitylogapp.EffectiveRelationIDs(currentPermissionIDs, permissionIDs, false)
	if len(removedPermissionIDs) == 0 {
		return nil
	}
	return r.tx.Do(ctx, func(txCtx context.Context) error {
		if err := r.roleRepo.RemovePermissions(txCtx, role.ID, removedPermissionIDs); err != nil {
			return shared.NewAppError(shared.ErrTypeInternal, "Failed to remove permissions from role", err)
		}
		return activitylogapp.RecordActivity(txCtx, r.activityLogRepo, activitylogdomain.ActivityRolePermissionsRemoved, activitylogdomain.ActivitySubjectRole, role.ID,
			activitylogapp.RelationChanges("permissions", []uuid.UUID{}, removedPermissionIDs))
	})
}
