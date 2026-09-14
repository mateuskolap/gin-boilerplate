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
	roleRepo           domain.RoleRepository
	rolePermissionRepo domain.RolePermissionRepository
}

func NewRoleUseCase(
	roleRepo domain.RoleRepository,
	rolePermissionRepo domain.RolePermissionRepository,
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
		roleRepo:           roleRepo,
		rolePermissionRepo: rolePermissionRepo,
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

	return r.invalidateRolePermissions(ctx, role.Name)
}

func (r *roleUseCase) Update(ctx context.Context, role *domain.Role) error {
	existingRole, err := r.Find(ctx, role.ID)
	if err != nil {
		return err
	}

	oldName := existingRole.Name
	existingRole.Name = role.Name

	if err = r.roleRepo.Update(ctx, existingRole); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to update role",
			err,
		)
	}

	if err := r.invalidateRolePermissions(ctx, oldName); err != nil {
		return err
	}
	if oldName != role.Name {
		if err := r.invalidateRolePermissions(ctx, role.Name); err != nil {
			return err
		}
	}

	return nil
}

func (r *roleUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	existingRole, err := r.Find(ctx, id)
	if err != nil {
		return err
	}

	if err := r.BaseDeleteUseCase.Delete(ctx, id); err != nil {
		return err
	}

	return r.invalidateRolePermissions(ctx, existingRole.Name)
}

func (r *roleUseCase) AddPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	role, err := r.Find(ctx, roleID)
	if err != nil {
		return err
	}

	if err := r.roleRepo.AddPermissions(ctx, *role, permissionIDs); err != nil {
		return err
	}

	return r.invalidateRolePermissions(ctx, role.Name)
}

func (r *roleUseCase) RemovePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	role, err := r.Find(ctx, roleID)
	if err != nil {
		return err
	}

	if err := r.roleRepo.RemovePermissions(ctx, *role, permissionIDs); err != nil {
		return err
	}

	return r.invalidateRolePermissions(ctx, role.Name)
}

func (r *roleUseCase) invalidateRolePermissions(ctx context.Context, roleName string) error {
	if err := r.rolePermissionRepo.InvalidatePermissionsByRole(ctx, roleName); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to invalidate role permissions",
			err,
		)
	}
	return nil
}
