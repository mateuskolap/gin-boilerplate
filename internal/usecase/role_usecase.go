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

	return nil
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

	_ = r.rolePermissionRepo.InvalidatePermissionsByRole(ctx, oldName)
	if oldName != role.Name {
		_ = r.rolePermissionRepo.InvalidatePermissionsByRole(ctx, role.Name)
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

	_ = r.rolePermissionRepo.InvalidatePermissionsByRole(ctx, existingRole.Name)
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

	_ = r.rolePermissionRepo.InvalidatePermissionsByRole(ctx, role.Name)
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

	_ = r.rolePermissionRepo.InvalidatePermissionsByRole(ctx, role.Name)
	return nil
}
