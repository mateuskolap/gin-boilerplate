package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/pkg/security"
	"time"

	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var allowedUserFilterFields = map[string]bool{
	"name":  true,
	"email": true,
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

func (u *userUseCase) Login(ctx context.Context, email string, password string) (string, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", domain.NewAppError(
			domain.ErrTypeInternal,
			"There was a problem verifying credentials",
			err,
		)
	}

	if user == nil {
		return "", domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid email or password",
			nil,
		)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid email or password",
			nil,
		)
	}

	return u.generateAccessToken(user.ID)
}

func (u *userUseCase) generateAccessToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ID:        uuid.New().String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(u.jwtExpiration)),
		NotBefore: jwt.NewNumericDate(now),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := jwtToken.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return "", domain.NewAppError(
			domain.ErrTypeInternal,
			"Error signing JWT token",
			err,
		)
	}

	return tokenString, nil
}

func (u *userUseCase) Logout(ctx context.Context, tokenString string) error {
	claims, err := security.ParseAndValidateJWT(tokenString, u.jwtSecret)
	if err != nil {
		return domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Invalid token",
			err,
		)
	}

	if claims.ExpiresAt == nil {
		return domain.NewAppError(
			domain.ErrTypeValidation,
			"Token without expiration cannot be revoked",
			nil,
		)
	}

	remainingTTL := time.Until(claims.ExpiresAt.Time)
	if remainingTTL <= 0 {
		return nil
	}

	if err := u.tokenBlacklist.RevokeToken(ctx, claims.ID, remainingTTL); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to revoke token",
			err,
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

func (u *userUseCase) Refresh(ctx context.Context, refreshToken string) (token string, err error) {
	storedToken, err := u.refreshTokenUseCase.Validate(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	user, err := u.Find(ctx, storedToken.UserID)
	if err != nil {
		return "", domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"User no longer exists",
			err,
		)
	}

	return u.generateAccessToken(user.ID)
}
