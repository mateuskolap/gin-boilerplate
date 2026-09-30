package application

import (
	"context"
	"errors"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	"gin-boilerplate/internal/infra/security"
	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
	roledomain "gin-boilerplate/internal/roles/domain"
	userdomain "gin-boilerplate/internal/users/domain"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"

	"golang.org/x/crypto/bcrypt"
)

var dummyPasswordHash = mustGenerateDummyPasswordHash()

const accountRateLimit = 5

func mustGenerateDummyPasswordHash() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("invalid-password-placeholder"), bcrypt.DefaultCost)
	if err != nil {
		panic("failed to initialize password comparison hash")
	}
	return hash
}

type authUseCase struct {
	userRepo                userdomain.UserRepository
	roleRepo                roledomain.RoleRepository
	refreshTokenUseCase     refreshtokendomain.RefreshTokenUseCase
	tokenBlacklist          userdomain.TokenBlackListRepository
	rateLimiter             port.RateLimiter
	tx                      port.TransactionManager
	jwtSecret               string
	jwtIssuer               string
	jwtAudience             string
	jwtExpiration           time.Duration
	passwordValidationLevel int
	passwordChecker         port.CompromisedPasswordChecker
}

func NewAuthUseCase(
	userRepo userdomain.UserRepository,
	roleRepo roledomain.RoleRepository,
	refreshTokenUseCase refreshtokendomain.RefreshTokenUseCase,
	tokenBlacklist userdomain.TokenBlackListRepository,
	tx port.TransactionManager,
	jwtSecret, jwtIssuer, jwtAudience string,
	jwtExpiration time.Duration,
	passwordValidationLevel int,
	passwordChecker port.CompromisedPasswordChecker,
	rateLimiter port.RateLimiter,
) userdomain.AuthUseCase {
	return &authUseCase{
		userRepo:                userRepo,
		roleRepo:                roleRepo,
		refreshTokenUseCase:     refreshTokenUseCase,
		tokenBlacklist:          tokenBlacklist,
		rateLimiter:             rateLimiter,
		tx:                      tx,
		jwtSecret:               jwtSecret,
		jwtIssuer:               jwtIssuer,
		jwtAudience:             jwtAudience,
		jwtExpiration:           jwtExpiration,
		passwordValidationLevel: passwordValidationLevel,
		passwordChecker:         passwordChecker,
	}
}

func (a *authUseCase) Register(ctx context.Context, user *userdomain.User) error {
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.Name = strings.TrimSpace(user.Name)
	if nameLength := utf8.RuneCountInString(user.Name); nameLength < 2 || nameLength > 100 {
		return shared.NewAppError(
			shared.ErrTypeValidation,
			"Name must contain between 2 and 100 characters",
			nil,
		)
	}
	if err := validatePassword(user.Password, a.passwordValidationLevel); err != nil {
		return err
	}
	if err := a.limitAccount(ctx, "register", user.Email, accountRateLimit); err != nil {
		return err
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
	if err := a.rejectPwnedPassword(ctx, user.Password); err != nil {
		return err
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

	defaultRole, err := a.roleRepo.GetByName(ctx, roledomain.RoleUser)
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
	user.Roles = []roledomain.Role{*defaultRole}

	if err := a.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, shared.ErrConflict) {
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

func (a *authUseCase) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*userdomain.AuthTokens, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := a.limitAccount(ctx, "login", email, accountRateLimit); err != nil {
		return nil, err
	}
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

	return &userdomain.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *authUseCase) limitAccount(ctx context.Context, action, identifier string, limit int) error {
	key := "rate_limit:account:" + action + ":" + security.HashSHA256(identifier)
	result, err := a.rateLimiter.Allow(ctx, key, limit, time.Minute)
	if err != nil {
		return shared.NewAppError(shared.ErrTypeUnavailable, "Rate limiting service is unavailable", err)
	}
	if !result.Allowed {
		return shared.NewAppError(shared.ErrTypeTooManyRequests, "Too many account attempts. Please try again later.", nil)
	}
	return nil
}

func (a *authUseCase) Refresh(ctx context.Context, refreshToken string, ipAddress, userAgent string) (*userdomain.AuthTokens, error) {
	storedToken, err := a.refreshTokenUseCase.Validate(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if err := a.limitAccount(ctx, "refresh", storedToken.UserID.String(), 30); err != nil {
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

	return &userdomain.AuthTokens{
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

func (a *authUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if err := validatePassword(newPassword, a.passwordValidationLevel); err != nil {
		return err
	}
	if currentPassword == newPassword {
		return shared.NewAppError(shared.ErrTypeValidation, "New password must differ from current password", nil)
	}

	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return shared.NewAppError(shared.ErrTypeInternal, "Failed to find user", err)
	}
	if user == nil {
		return shared.NewAppError(shared.ErrTypeUnauthorized, "Invalid user", nil)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return shared.NewAppError(shared.ErrTypeUnauthorized, "Current password is invalid", nil)
	}
	if err := a.rejectPwnedPassword(ctx, newPassword); err != nil {
		return err
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return shared.NewAppError(shared.ErrTypeInternal, "Failed to generate password hash", err)
	}

	if err := a.tokenBlacklist.RevokeUserTokens(ctx, userID.String(), a.jwtExpiration); err != nil {
		return shared.NewAppError(shared.ErrTypeInternal, "Failed to invalidate access tokens", err)
	}

	if err := a.tx.Do(ctx, func(txCtx context.Context) error {
		existingUser, err := a.userRepo.GetByID(txCtx, userID)
		if err != nil {
			return shared.NewAppError(shared.ErrTypeInternal, "Failed to find user", err)
		}
		if existingUser == nil {
			return shared.NewAppError(shared.ErrTypeUnauthorized, "Invalid user", nil)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(currentPassword)); err != nil {
			return shared.NewAppError(shared.ErrTypeUnauthorized, "Current password is invalid", nil)
		}

		existingUser.Password = string(newPasswordHash)
		if err := a.userRepo.Update(txCtx, existingUser); err != nil {
			return shared.NewAppError(shared.ErrTypeInternal, "Failed to update password", err)
		}
		if err := a.refreshTokenUseCase.RevokeAllByUserID(txCtx, userID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (a *authUseCase) ValidateAccessToken(ctx context.Context, tokenString string) (*userdomain.TokenClaims, error) {
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

	return &userdomain.TokenClaims{
		Subject: claims.Subject,
		TokenID: claims.ID,
	}, nil
}

func validatePassword(password string, level int) error {
	if utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return shared.NewAppError(
			shared.ErrTypeValidation,
			"Password must contain at least 8 characters and at most 72 bytes",
			nil,
		)
	}
	if level < 2 {
		return nil
	}

	var upper, lower, digit, symbol bool
	for _, char := range password {
		upper = upper || unicode.IsUpper(char)
		lower = lower || unicode.IsLower(char)
		digit = digit || unicode.IsDigit(char)
		symbol = symbol || unicode.IsPunct(char) || unicode.IsSymbol(char)
	}
	if !upper || !lower || !digit || !symbol {
		return shared.NewAppError(
			shared.ErrTypeValidation,
			"Password must contain uppercase and lowercase letters, a number, and a symbol",
			nil,
		)
	}
	return nil
}

func (a *authUseCase) rejectPwnedPassword(ctx context.Context, password string) error {
	if a.passwordValidationLevel < 3 {
		return nil
	}
	pwned, err := a.passwordChecker.IsCompromised(ctx, password)
	if err != nil || !pwned {
		return nil
	}
	return shared.NewAppError(shared.ErrTypeValidation, "Password has appeared in a data breach", nil)
}
