package application

import (
	"context"

	permissiondomain "gin-boilerplate/internal/permissions/domain"

	"uuid"
)

type permissionCheckerUseCase struct {
	authorizationRepo permissiondomain.AuthorizationRepository
}

func NewPermissionCheckerUseCase(
	authorizationRepo permissiondomain.AuthorizationRepository,
) permissiondomain.PermissionCheckerUseCase {
	return &permissionCheckerUseCase{authorizationRepo: authorizationRepo}
}

func (p *permissionCheckerUseCase) HasPermission(
	ctx context.Context,
	userID uuid.UUID,
	permission permissiondomain.PermissionName,
) (bool, error) {
	return p.authorizationRepo.UserHasPermission(ctx, userID, permission)
}
