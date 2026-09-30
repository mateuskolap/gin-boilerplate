package domain

import (
	"context"

	"uuid"
)

type AuthorizationRepository interface {
	// UserHasPermission checks the current database state. Authorization data is
	// deliberately not copied into access tokens or a distributed cache.
	UserHasPermission(ctx context.Context, userID uuid.UUID, permission PermissionName) (bool, error)
}

type PermissionCheckerUseCase interface {
	// HasPermission checks whether the user currently has the named permission.
	HasPermission(ctx context.Context, userID uuid.UUID, permission PermissionName) (bool, error)
}
