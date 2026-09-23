package usecase

import (
	"context"

	"gin-boilerplate/internal/domain"

	"uuid"
)

type permissionCheckerUseCase struct {
	authorizationRepo domain.AuthorizationRepository
}

func NewPermissionCheckerUseCase(
	authorizationRepo domain.AuthorizationRepository,
) domain.PermissionCheckerUseCase {
	return &permissionCheckerUseCase{authorizationRepo: authorizationRepo}
}

func (p *permissionCheckerUseCase) HasPermission(
	ctx context.Context,
	userID uuid.UUID,
	permission domain.PermissionName,
) (bool, error) {
	return p.authorizationRepo.UserHasPermission(ctx, userID, permission)
}
