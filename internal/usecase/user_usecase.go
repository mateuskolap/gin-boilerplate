package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	"strings"
	"time"
	"unicode/utf8"

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
	shared.BaseListUseCase[domain.User]
	shared.BaseFindUseCase[domain.User]
	userRepo            domain.UserRepository
	refreshTokenUseCase domain.RefreshTokenUseCase
	storage             port.Storage
	tokenBlacklist      domain.TokenBlackListRepository
	tx                  port.TransactionManager
	jwtExpiration       time.Duration
}

func NewUserUseCase(
	userRepo domain.UserRepository,
	refreshTokenUseCase domain.RefreshTokenUseCase,
	storage port.Storage,
	tokenBlacklist domain.TokenBlackListRepository,
	tx port.TransactionManager,
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
		storage:             storage,
		tokenBlacklist:      tokenBlacklist,
		tx:                  tx,
		jwtExpiration:       jwtExpiration,
	}
}

func (u *userUseCase) UpdateProfile(ctx context.Context, user *domain.User) error {
	user.Name = strings.TrimSpace(user.Name)
	if nameLength := utf8.RuneCountInString(user.Name); nameLength < 2 || nameLength > 100 {
		return shared.NewAppError(
			shared.ErrTypeValidation,
			"Name must contain between 2 and 100 characters",
			nil,
		)
	}
	existingUser, err := findByID(ctx, u.userRepo, user.ID)
	if err != nil {
		return err
	}

	existingUser.Name = user.Name

	if err := u.userRepo.Update(ctx, existingUser); err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to update profile",
			err,
		)
	}

	*user = *existingUser
	return nil
}

func (u *userUseCase) Delete(ctx context.Context, userID uuid.UUID) error {
	existingUser, err := findByID(ctx, u.userRepo, userID)
	if err != nil {
		return err
	}

	err = u.tx.Do(ctx, func(txCtx context.Context) error {
		if err := u.refreshTokenUseCase.RevokeAllByUserID(txCtx, userID); err != nil {
			return shared.NewAppError(
				shared.ErrTypeInternal,
				"Failed to revoke all refresh tokens for user",
				err,
			)
		}

		if err := u.userRepo.Delete(txCtx, existingUser.ID); err != nil {
			return shared.NewAppError(
				shared.ErrTypeInternal,
				"Failed to delete user",
				err,
			)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := u.tokenBlacklist.RevokeUserTokens(ctx, userID.String(), u.jwtExpiration); err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to revoke user tokens",
			err,
		)
	}

	return nil
}

func (u *userUseCase) AddRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	user, err := findByID(ctx, u.userRepo, userID)
	if err != nil {
		return err
	}

	if err := u.userRepo.AddRoles(ctx, *user, roleIDs); err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to add roles to user",
			err,
		)
	}

	if err := u.tokenBlacklist.RevokeUserTokens(ctx, userID.String(), u.jwtExpiration); err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to revoke user tokens",
			err,
		)
	}

	return nil
}

func (u *userUseCase) RemoveRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	user, err := findByID(ctx, u.userRepo, userID)
	if err != nil {
		return err
	}

	if err := u.userRepo.RemoveRoles(ctx, *user, roleIDs); err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to remove roles from user",
			err,
		)
	}

	if err := u.tokenBlacklist.RevokeUserTokens(ctx, userID.String(), u.jwtExpiration); err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to revoke user tokens",
			err,
		)
	}

	return nil
}

func (u *userUseCase) UpdateImage(ctx context.Context, userID uuid.UUID, file shared.UploadedFile) error {
	existingUser, err := findByID(ctx, u.userRepo, userID)
	if err != nil {
		return err
	}

	if existingUser.AvatarKey != "" {
		if err := u.storage.Delete(ctx, )
	}
}

func (u *userUseCase) RemoveImage(ctx context.Context, userID uuid.UUID) error {
	panic("uninplemented")
}
