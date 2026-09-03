package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/pkg"
	"gin-boilerplate/pkg/security"
	"time"

	"uuid"

	"golang.org/x/crypto/bcrypt"
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
	tokenBlacklist      domain.TokenBlackList
	jwtSecret           string
	jwtExpiration       time.Duration
}

func NewUserUseCase(
	userRepo domain.UserRepository,
	refreshTokenUseCase domain.RefreshTokenUseCase,
	tokenBlacklist domain.TokenBlackList,
	jwtSecret string,
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
		jwtSecret:           jwtSecret,
		jwtExpiration:       jwtExpiration,
	}
}

func (u *userUseCase) Register(ctx context.Context, user *domain.User) error {
	existingUser, err := u.userRepo.GetByEmail(ctx, user.Email)
	if err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"There was a problem verifying the email",
			err,
		)
	}

	if existingUser != nil {
		return domain.NewAppError(
			domain.ErrTypeConflict,
			"This email is already in use",
			nil,
		)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Error while generating the password hash",
			err,
		)
	}

	user.Password = string(hashedPassword)

	if err := u.userRepo.Create(ctx, user); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to create user",
			err,
		)
	}

	return nil
}

func (u *userUseCase) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*domain.AuthTokens, error) {
	user, err := u.userRepo.GetByEmail(ctx, email, "Roles")
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"There was a problem verifying credentials",
			err,
		)
	}

	if user == nil {
		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid email or password",
			nil,
		)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid email or password",
			nil,
		)
	}

	roles := pkg.Map(user.Roles, func(role domain.Role) string { return role.Name })

	accessToken, err := security.GenerateAccessToken(user.ID, roles, u.jwtSecret, u.jwtExpiration)
	if err != nil {
		return nil, err
	}

	refreshToken, err := u.refreshTokenUseCase.Create(ctx, user.ID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	authTokens := domain.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return &authTokens, nil
}

func (u *userUseCase) Refresh(ctx context.Context, refreshToken string, ipAddress, userAgent string) (*domain.AuthTokens, error) {
	storedToken, err := u.refreshTokenUseCase.Validate(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := u.Find(ctx, storedToken.UserID)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"User no longer exists",
			err,
		)
	}

	roles := pkg.Map(user.Roles, func(role domain.Role) string { return role.Name })

	accessToken, err := security.GenerateAccessToken(user.ID, roles, u.jwtSecret, u.jwtExpiration)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := u.refreshTokenUseCase.Rotate(ctx, refreshToken, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	authTokens := domain.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}

	return &authTokens, nil
}

func (u *userUseCase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	var tokenErr error

	if accessToken != "" {
		claims, err := security.ParseAndValidateJWT(accessToken, u.jwtSecret)
		if err != nil {
			tokenErr = err
		} else if claims != nil && claims.ExpiresAt != nil {
			remainingTTL := time.Until(claims.ExpiresAt.Time)
			if remainingTTL > 0 {
				if err := u.tokenBlacklist.RevokeToken(ctx, claims.ID, remainingTTL); err != nil {
					return domain.NewAppError(
						domain.ErrTypeInternal,
						"Failed to revoke access token",
						err,
					)
				}
			}
		}
	}

	if refreshToken != "" {
		if err := u.refreshTokenUseCase.Revoke(ctx, refreshToken); err != nil {
			return err
		}
	} else if tokenErr != nil {
		return domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid token",
			tokenErr,
		)
	}

	return nil
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
