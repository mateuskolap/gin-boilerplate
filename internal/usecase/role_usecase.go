package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"

	"uuid"
)

var allowedRoleFilterFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
	"id":         true,
}

type roleUseCase struct {
	domain.BaseListUseCase[domain.Role]
	domain.BaseFindUseCase[domain.Role]
	domain.BaseDeleteUseCase
	roleRepo domain.RoleRepository
}

func NewRoleUseCase(roleRepo domain.RoleRepository) domain.RoleUseCase {
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
	existingRole, err := r.roleRepo.GetByName(ctx, role.Name)
	if err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"There was a problem verifying the role name",
			err,
		)
	}

	if existingRole != nil {
		return domain.NewAppError(
			domain.ErrTypeConflict,
			"This role already exists",
			nil,
		)
	}

	if err := r.roleRepo.Create(ctx, role); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to create role",
			err,
		)
	}

	return nil
}

func (r *roleUseCase) Update(ctx context.Context, role *domain.Role) error {
	existingRole, err := r.Find(ctx, role.ID)
	if err != nil {
		return err
	}

	existingRole.Name = role.Name

	if err = r.roleRepo.Update(ctx, existingRole); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to update role",
			err,
		)
	}

	return nil
}

func (r *roleUseCase) AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	role, err := r.Find(ctx, roleID)
	if err != nil {
		return err
	}

	return r.roleRepo.AddPermissions(ctx, *role, permissionIDs)
}

func (r *roleUseCase) RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	role, err := r.Find(ctx, roleID)
	if err != nil {
		return err
	}

	return r.roleRepo.RemovePermissions(ctx, *role, permissionIDs)
}
