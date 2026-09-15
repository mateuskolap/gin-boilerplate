package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/infra/security"
	"gin-boilerplate/pkg/collection"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type authUseCase struct {
	userRepo            domain.UserRepository
	roleRepo            domain.RoleRepository
	refreshTokenUseCase domain.RefreshTokenUseCase
	tokenBlacklist      domain.TokenBlackListRepository
	jwtSecret           string
	jwtExpiration       time.Duration
}

func NewAuthUseCase(
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
	refreshTokenUseCase domain.RefreshTokenUseCase,
	tokenBlacklist domain.TokenBlackListRepository,
	jwtSecret string,
	jwtExpiration time.Duration,
) domain.AuthUseCase {
	return &authUseCase{
		userRepo:            userRepo,
		roleRepo:            roleRepo,
		refreshTokenUseCase: refreshTokenUseCase,
		tokenBlacklist:      tokenBlacklist,
		jwtSecret:           jwtSecret,
		jwtExpiration:       jwtExpiration,
	}
}

func (a *authUseCase) Register(ctx context.Context, user *domain.User) error {
	existingUser, err := a.userRepo.GetByEmail(ctx, user.Email)
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

	defaultRole, err := a.roleRepo.GetByName(ctx, domain.RoleUser)
	if err == nil && defaultRole != nil {
		user.Roles = []domain.Role{*defaultRole}
	}

	if err := a.userRepo.Create(ctx, user); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to create user",
			err,
		)
	}

	return nil
}

func (a *authUseCase) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*domain.AuthTokens, error) {
	user, err := a.userRepo.GetByEmail(ctx, email, "Roles")
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

	roles := collection.Map(user.Roles, func(role domain.Role) string { return role.Name })

	accessToken, err := security.GenerateAccessToken(user.ID, roles, a.jwtSecret, a.jwtExpiration)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to generate access token",
			err,
		)
	}

	refreshToken, err := a.refreshTokenUseCase.Create(ctx, user.ID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	return &domain.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *authUseCase) Refresh(ctx context.Context, refreshToken string, ipAddress, userAgent string) (*domain.AuthTokens, error) {
	storedToken, err := a.refreshTokenUseCase.Validate(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := a.userRepo.GetByID(ctx, storedToken.UserID, "Roles")
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to find user",
			err,
		)
	}

	if user == nil {
		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"User no longer exists",
			nil,
		)
	}

	roles := collection.Map(user.Roles, func(role domain.Role) string { return role.Name })

	accessToken, err := security.GenerateAccessToken(user.ID, roles, a.jwtSecret, a.jwtExpiration)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to generate access token",
			err,
		)
	}

	newRefreshToken, err := a.refreshTokenUseCase.Rotate(ctx, refreshToken, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	return &domain.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (a *authUseCase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	var tokenErr error

	if accessToken != "" {
		claims, err := security.ParseAndValidateJWT(accessToken, a.jwtSecret)
		if err != nil {
			tokenErr = err
		} else if claims != nil && claims.ExpiresAt != nil {
			remainingTTL := time.Until(claims.ExpiresAt.Time)
			if remainingTTL > 0 {
				if err := a.tokenBlacklist.RevokeToken(ctx, claims.ID, remainingTTL); err != nil {
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
		if err := a.refreshTokenUseCase.Revoke(ctx, refreshToken); err != nil {
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

func (a *authUseCase) ValidateAccessToken(ctx context.Context, tokenString string) (*domain.TokenClaims, error) {
	claims, err := security.ParseAndValidateJWT(tokenString, a.jwtSecret)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid or expired token",
			err,
		)
	}

	isRevoked, err := a.tokenBlacklist.IsRevoked(ctx, claims.ID)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to check token revocation status",
			err,
		)
	}
	if isRevoked {
		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid or expired token",
			nil,
		)
	}

	var issuedAt time.Time
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time
	}

	isUserRevoked, err := a.tokenBlacklist.IsUserTokenRevoked(ctx, claims.Subject, issuedAt)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to check user token revocation status",
			err,
		)
	}
	if isUserRevoked {
		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid or expired token",
			nil,
		)
	}

	return &domain.TokenClaims{
		Subject: claims.Subject,
		TokenID: claims.ID,
		Roles:   claims.Roles,
	}, nil
}
