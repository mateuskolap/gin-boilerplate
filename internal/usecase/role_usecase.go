package usecase

import (
	"context"
	"errors"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
	"uuid"
)

var allowedRoleFilterFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
	"id":         true,
}

type roleUseCase struct {
	shared.BaseListUseCase[domain.Role]
	shared.BaseFindUseCase[domain.Role]
	shared.BaseDeleteUseCase
	roleRepo domain.RoleRepository
}

func NewRoleUseCase(
	roleRepo domain.RoleRepository,
) domain.RoleUseCase {
	return &roleUseCase{
		BaseListUseCase: NewBaseListUseCase(
			roleRepo,
			allowedRoleFilterFields,
		),
		BaseFindUseCase: NewBaseFindUseCase(
			roleRepo,
			"Permissions",
		),
		BaseDeleteUseCase: NewBaseDeleteUseCase(
			roleRepo,
		),
		roleRepo: roleRepo,
	}
}

func (r *roleUseCase) Create(ctx context.Context, role *domain.Role) error {
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
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return shared.NewAppError(
				shared.ErrTypeConflict,
				"This role already exists",
				err,
			)
		}
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to create role",
			err,
		)
	}

	return nil
}

func (r *roleUseCase) Update(ctx context.Context, role *domain.Role) error {
	if err := normalizeRoleName(role); err != nil {
		return err
	}
	existingRole, err := r.Find(ctx, role.ID)
	if err != nil {
		return err
	}

	existingRole.Name = role.Name

	if err = r.roleRepo.Update(ctx, existingRole); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return shared.NewAppError(
				shared.ErrTypeConflict,
				"This role already exists",
				err,
			)
		}
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to update role",
			err,
		)
	}

	*role = *existingRole
	return nil
}

func normalizeRoleName(role *domain.Role) error {
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

func (r *roleUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := r.Find(ctx, id); err != nil {
		return err
	}

	if err := r.BaseDeleteUseCase.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (r *roleUseCase) AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	role, err := r.Find(ctx, roleID)
	if err != nil {
		return err
	}

	if err := r.roleRepo.AddPermissions(ctx, *role, permissionIDs); err != nil {
		return err
	}

	return nil
}

func (r *roleUseCase) RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	role, err := r.Find(ctx, roleID)
	if err != nil {
		return err
	}

	if err := r.roleRepo.RemovePermissions(ctx, *role, permissionIDs); err != nil {
		return err
	}

	return nil
}
