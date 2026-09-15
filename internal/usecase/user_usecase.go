package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"time"

	"uuid"
)

var allowedUserFilterFields = map[string]bool{
	"name":       true,
	"email":      true,
	"created_at": true,
	"updated_at": true,
	"id":         true,
}

type userUseCase struct {
	domain.BaseListUseCase[domain.User]
	domain.BaseFindUseCase[domain.User]
	userRepo            domain.UserRepository
	refreshTokenUseCase domain.RefreshTokenUseCase
	tokenBlacklist      domain.TokenBlackListRepository
	jwtExpiration       time.Duration
}

func NewUserUseCase(
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
	refreshTokenUseCase domain.RefreshTokenUseCase,
	tokenBlacklist domain.TokenBlackListRepository,
	jwtExpiration time.Duration,
) domain.UserUseCase {
	return &userUseCase{
		BaseListUseCase: NewBaseListUseCase(
			userRepo,
			allowedUserFilterFields,
		),
		BaseFindUseCase: NewBaseFindUseCase(
			userRepo,
			"Roles",
		),
		userRepo:            userRepo,
		refreshTokenUseCase: refreshTokenUseCase,
		tokenBlacklist:      tokenBlacklist,
		jwtExpiration:       jwtExpiration,
	}
}

func (u *userUseCase) UpdateProfile(ctx context.Context, user *domain.User) error {
	existingUser, err := u.Find(ctx, user.ID)
	if err != nil {
		return err
	}

	existingUser.Name = user.Name

	if err := u.userRepo.Update(ctx, existingUser); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to update profile",
			err,
		)
	}

	return nil
}

func (u *userUseCase) Delete(ctx context.Context, userID uuid.UUID) error {
	existingUser, err := u.Find(ctx, userID)
	if err != nil {
		return err
	}

	if err := u.refreshTokenUseCase.RevokeAllByUserID(ctx, userID); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to revoke all refresh tokens for user",
			err,
		)
	}

	if err := u.tokenBlacklist.RevokeUserTokens(ctx, userID.String(), u.jwtExpiration); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to revoke user tokens",
			err,
		)
	}

	if err := u.userRepo.Delete(ctx, existingUser.ID); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to delete user",
			err,
		)
	}

	return nil
}

func (u *userUseCase) AddRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	user, err := u.Find(ctx, userID)
	if err != nil {
		return err
	}

	if err := u.userRepo.AddRoles(ctx, *user, roleIDs); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to add roles to user",
			err,
		)
	}

	if err := u.tokenBlacklist.RevokeUserTokens(ctx, userID.String(), u.jwtExpiration); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to revoke user tokens",
			err,
		)
	}

	return nil
}

func (u *userUseCase) RemoveRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	user, err := u.Find(ctx, userID)
	if err != nil {
		return err
	}

	if err := u.userRepo.RemoveRoles(ctx, *user, roleIDs); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to remove roles from user",
			err,
		)
	}

	if err := u.tokenBlacklist.RevokeUserTokens(ctx, userID.String(), u.jwtExpiration); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to revoke user tokens",
			err,
		)
	}

	return nil
}
