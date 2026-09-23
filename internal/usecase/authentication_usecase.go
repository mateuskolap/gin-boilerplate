package usecase

import (
	"context"
	"errors"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"gin-boilerplate/internal/infra/security"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var dummyPasswordHash = mustGenerateDummyPasswordHash()

func mustGenerateDummyPasswordHash() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("invalid-password-placeholder"), bcrypt.DefaultCost)
	if err != nil {
		panic("failed to initialize password comparison hash")
	}
	return hash
}

type authUseCase struct {
	userRepo            domain.UserRepository
	roleRepo            domain.RoleRepository
	refreshTokenUseCase domain.RefreshTokenUseCase
	tokenBlacklist      domain.TokenBlackListRepository
	jwtSecret           string
	jwtIssuer           string
	jwtAudience         string
	jwtExpiration       time.Duration
}

func NewAuthUseCase(
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
	refreshTokenUseCase domain.RefreshTokenUseCase,
	tokenBlacklist domain.TokenBlackListRepository,
	jwtSecret, jwtIssuer, jwtAudience string,
	jwtExpiration time.Duration,
) domain.AuthUseCase {
	return &authUseCase{
		userRepo:            userRepo,
		roleRepo:            roleRepo,
		refreshTokenUseCase: refreshTokenUseCase,
		tokenBlacklist:      tokenBlacklist,
		jwtSecret:           jwtSecret,
		jwtIssuer:           jwtIssuer,
		jwtAudience:         jwtAudience,
		jwtExpiration:       jwtExpiration,
	}
}

func (a *authUseCase) Register(ctx context.Context, user *domain.User) error {
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.Name = strings.TrimSpace(user.Name)
	if nameLength := utf8.RuneCountInString(user.Name); nameLength < 2 || nameLength > 100 {
		return shared.NewAppError(
			shared.ErrTypeValidation,
			"Name must contain between 2 and 100 characters",
			nil,
		)
	}
	if len(user.Password) < 8 || len(user.Password) > 72 {
		return shared.NewAppError(
			shared.ErrTypeValidation,
			"Password must contain between 8 and 72 bytes",
			nil,
		)
	}

	existingUser, err := a.userRepo.GetByEmail(ctx, user.Email)
	if err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"There was a problem verifying the email",
			err,
		)
	}

	if existingUser != nil {
		return shared.NewAppError(
			shared.ErrTypeConflict,
			"This email is already in use",
			nil,
		)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Error while generating the password hash",
			err,
		)
	}

	user.Password = string(hashedPassword)

	defaultRole, err := a.roleRepo.GetByName(ctx, domain.RoleUser)
	if err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to find the default role",
			err,
		)
	}
	if defaultRole == nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Default role is not configured",
			nil,
		)
	}
	user.Roles = []domain.Role{*defaultRole}

	if err := a.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return shared.NewAppError(
				shared.ErrTypeConflict,
				"This email is already in use",
				err,
			)
		}
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to create user",
			err,
		)
	}

	return nil
}

func (a *authUseCase) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*domain.AuthTokens, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := a.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"There was a problem verifying credentials",
			err,
		)
	}

	if user == nil {
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))

		return nil, shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Invalid email or password",
			nil,
		)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Invalid email or password",
			nil,
		)
	}

	accessToken, err := security.GenerateAccessToken(
		user.ID,
		a.jwtSecret,
		a.jwtIssuer,
		a.jwtAudience,
		a.jwtExpiration,
	)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
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

	user, err := a.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to find user",
			err,
		)
	}

	if user == nil {
		return nil, shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"User no longer exists",
			nil,
		)
	}

	accessToken, err := security.GenerateAccessToken(
		user.ID,
		a.jwtSecret,
		a.jwtIssuer,
		a.jwtAudience,
		a.jwtExpiration,
	)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
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
		claims, err := security.ParseAndValidateJWT(accessToken, a.jwtSecret, a.jwtIssuer, a.jwtAudience)
		if err != nil {
			tokenErr = err
		} else {
			remainingTTL := time.Until(claims.ExpiresAt.Time)
			if remainingTTL > 0 {
				if err := a.tokenBlacklist.RevokeToken(ctx, claims.ID, remainingTTL); err != nil {
					return shared.NewAppError(
						shared.ErrTypeInternal,
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
		return shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Invalid token",
			tokenErr,
		)
	}

	return nil
}

func (a *authUseCase) ValidateAccessToken(ctx context.Context, tokenString string) (*domain.TokenClaims, error) {
	claims, err := security.ParseAndValidateJWT(tokenString, a.jwtSecret, a.jwtIssuer, a.jwtAudience)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Invalid or expired token",
			err,
		)
	}

	isRevoked, err := a.tokenBlacklist.IsRevoked(ctx, claims.ID)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to check token revocation status",
			err,
		)
	}
	if isRevoked {
		return nil, shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Invalid or expired token",
			nil,
		)
	}

	isUserRevoked, err := a.tokenBlacklist.IsUserTokenRevoked(ctx, claims.Subject, claims.IssuedAt.Time)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to check user token revocation status",
			err,
		)
	}
	if isUserRevoked {
		return nil, shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Invalid or expired token",
			nil,
		)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Invalid or expired token",
			err,
		)
	}
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to validate token owner",
			err,
		)
	}
	if user == nil {
		return nil, shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Invalid or expired token",
			nil,
		)
	}

	return &domain.TokenClaims{
		Subject: claims.Subject,
		TokenID: claims.ID,
	}, nil
}
