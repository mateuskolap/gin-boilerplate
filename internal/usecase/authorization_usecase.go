package usecase

import (
	"context"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"

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
	allowed, err := p.authorizationRepo.UserHasPermission(ctx, userID, permission)
	if err != nil {
		return false, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to check permission",
			err,
		)
	}
	return allowed, nil
}
