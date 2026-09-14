package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"

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
	userRepo domain.UserRepository
	roleRepo domain.RoleRepository
}

func NewUserUseCase(
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
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
		userRepo: userRepo,
		roleRepo: roleRepo,
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

func (u *userUseCase) AddRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	user, err := u.Find(ctx, userID)
	if err != nil {
		return err
	}

	return u.userRepo.AddRoles(ctx, *user, roleIDs)
}

func (u *userUseCase) RemoveRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	user, err := u.Find(ctx, userID)
	if err != nil {
		return err
	}

	return u.userRepo.RemoveRoles(ctx, *user, roleIDs)
}
