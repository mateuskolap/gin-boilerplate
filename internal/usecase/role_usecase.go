package usecase

import (
	"context"
	"fmt"
	"gin-boilerplate/internal/domain"

	"github.com/google/uuid"
)

var allowedRoleFilterFields = map[string]bool{
	"name": true,
}

type roleUseCase struct {
	roleRepo domain.RoleRepository
}

func NewRoleUseCase(roleRepo domain.RoleRepository) domain.RoleUseCase {
	return &roleUseCase{
		roleRepo: roleRepo,
	}
}

func (r *roleUseCase) Create(ctx context.Context, role *domain.Role) error {
	existingRole, err := r.roleRepo.GetByName(ctx, role.Name)
	if err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"There was a poblem verifying the role name",
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

func (r *roleUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.roleRepo.Delete(ctx, id); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to delete role",
			err,
		)
	}
	return nil
}

func (r *roleUseCase) Find(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	role, err := r.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to retrieve role",
			err,
		)
	}

	if role == nil {
		return nil, domain.NewAppError(
			domain.ErrTypeNotFound,
			"Role not found",
			nil,
		)
	}

	return role, nil
}

func (r *roleUseCase) List(
	ctx context.Context,
	params domain.PaginationParams,
	filters []domain.Filter,
) (*domain.PaginatedResult[domain.Role], error) {
	for _, f := range filters {
		if !allowedRoleFilterFields[f.Field] {
			return nil, domain.NewAppError(
				domain.ErrTypeValidation,
				fmt.Sprintf("filtering by field '%s' is not allowed", f.Field),
				nil,
			)
		}
	}

	result, err := r.roleRepo.List(ctx, params, filters)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to retrieve roles",
			err,
		)
	}

	return result, nil
}

func (r *roleUseCase) Update(ctx context.Context, role *domain.Role) error {
	existingRole, err := r.roleRepo.GetByID(ctx, role.ID)
	if err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to retrieve existing role",
			err,
		)
	}

	if existingRole == nil {
		return domain.NewAppError(
			domain.ErrTypeNotFound,
			"Role not found",
			nil,
		)
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
