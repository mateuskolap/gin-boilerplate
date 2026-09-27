package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"gin-boilerplate/internal/infra/security"

	"golang.org/x/crypto/bcrypt"
	"uuid"
)

type immediateTransactionManager struct{}

func (immediateTransactionManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type refreshTokenRepositoryFake struct {
	tokens map[uuid.UUID]*domain.RefreshToken
}

func (r *refreshTokenRepositoryFake) Create(_ context.Context, token *domain.RefreshToken) error {
	r.tokens[token.ID] = token
	return nil
}

func (r *refreshTokenRepositoryFake) GetByID(_ context.Context, id uuid.UUID, _ ...string) (*domain.RefreshToken, error) {
	return r.tokens[id], nil
}

func (*refreshTokenRepositoryFake) FindOneBy(context.Context, string, []any, ...string) (*domain.RefreshToken, error) {
	return nil, nil
}

func (r *refreshTokenRepositoryFake) Update(_ context.Context, token *domain.RefreshToken) error {
	r.tokens[token.ID] = token
	return nil
}

func (r *refreshTokenRepositoryFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.tokens, id)
	return nil
}

func (*refreshTokenRepositoryFake) List(context.Context, shared.PaginationParams, []shared.Filter, ...string) (*shared.PaginatedResult[domain.RefreshToken], error) {
	return &shared.PaginatedResult[domain.RefreshToken]{}, nil
}

func (r *refreshTokenRepositoryFake) FindByTokenHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	for _, token := range r.tokens {
		if token.TokenHash == hash {
			return token, nil
		}
	}
	return nil, nil
}

func (*refreshTokenRepositoryFake) ListByUserID(context.Context, uuid.UUID, shared.PaginationParams, []shared.Filter) (*shared.PaginatedResult[domain.RefreshToken], error) {
	return &shared.PaginatedResult[domain.RefreshToken]{}, nil
}

func (r *refreshTokenRepositoryFake) RevokeAllByUserID(_ context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	for _, token := range r.tokens {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &now
		}
	}
	return nil
}

func (r *refreshTokenRepositoryFake) RevokeByID(_ context.Context, id, replacedBy uuid.UUID) (bool, error) {
	token := r.tokens[id]
	if token == nil || token.RevokedAt != nil {
		return false, nil
	}
	now := time.Now().UTC()
	token.RevokedAt = &now
	token.ReplacedBy = &replacedBy
	return true, nil
}

func (r *refreshTokenRepositoryFake) RevokeActiveByIDAndUserID(_ context.Context, id, userID uuid.UUID) (bool, error) {
	token := r.tokens[id]
	if token == nil || token.UserID != userID || token.RevokedAt != nil || !token.ExpiresAt.After(time.Now().UTC()) {
		return false, nil
	}
	now := time.Now().UTC()
	token.RevokedAt = &now
	return true, nil
}

func (r *refreshTokenRepositoryFake) RevokeAllByUserIDExceptID(_ context.Context, userID, exceptID uuid.UUID) (int64, error) {
	var revoked int64
	now := time.Now().UTC()
	for _, token := range r.tokens {
		if token.UserID == userID && token.ID != exceptID && token.RevokedAt == nil && token.ExpiresAt.After(now) {
			token.RevokedAt = &now
			revoked++
		}
	}
	return revoked, nil
}

func (r *refreshTokenRepositoryFake) DeleteExpiredBefore(_ context.Context, cutoff time.Time) (int64, error) {
	var deleted int64
	for id, token := range r.tokens {
		if token.ExpiresAt.Before(cutoff) {
			delete(r.tokens, id)
			deleted++
		}
	}
	return deleted, nil
}

type userRepositoryFake struct {
	users map[uuid.UUID]*domain.User
}

func (r *userRepositoryFake) Create(_ context.Context, user *domain.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *userRepositoryFake) GetByID(_ context.Context, id uuid.UUID, _ ...string) (*domain.User, error) {
	return r.users[id], nil
}

func (*userRepositoryFake) FindOneBy(context.Context, string, []any, ...string) (*domain.User, error) {
	return nil, nil
}

func (r *userRepositoryFake) Update(_ context.Context, user *domain.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *userRepositoryFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.users, id)
	return nil
}

func (*userRepositoryFake) List(context.Context, shared.PaginationParams, []shared.Filter, ...string) (*shared.PaginatedResult[domain.User], error) {
	return &shared.PaginatedResult[domain.User]{}, nil
}

func (r *userRepositoryFake) GetByEmail(_ context.Context, email string, _ ...string) (*domain.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

func (*userRepositoryFake) AddRoles(context.Context, domain.User, []uuid.UUID) error { return nil }
func (*userRepositoryFake) RemoveRoles(context.Context, domain.User, []uuid.UUID) error {
	return nil
}

type tokenBlacklistRepositoryFake struct {
	revokedUsers []string
}

func (*tokenBlacklistRepositoryFake) RevokeToken(context.Context, string, time.Duration) error {
	return nil
}

func (*tokenBlacklistRepositoryFake) IsRevoked(context.Context, string) (bool, error) {
	return false, nil
}

func (r *tokenBlacklistRepositoryFake) RevokeUserTokens(_ context.Context, userID string, _ time.Duration) error {
	r.revokedUsers = append(r.revokedUsers, userID)
	return nil
}

func (*tokenBlacklistRepositoryFake) IsUserTokenRevoked(context.Context, string, time.Time) (bool, error) {
	return false, nil
}

func TestRevokeOtherSessionsPreservesCurrentSession(t *testing.T) {
	userID := uuid.New()
	current := newRefreshToken(userID, "current")
	other := newRefreshToken(userID, "other")
	foreign := newRefreshToken(uuid.New(), "foreign")
	repo := &refreshTokenRepositoryFake{tokens: map[uuid.UUID]*domain.RefreshToken{
		current.ID: current,
		other.ID:   other,
		foreign.ID: foreign,
	}}
	useCase := NewRefreshTokenUseCase(repo, immediateTransactionManager{}, time.Hour)

	revoked, err := useCase.RevokeOtherSessions(context.Background(), userID, "current")
	if err != nil {
		t.Fatalf("RevokeOtherSessions() error = %v", err)
	}
	if revoked != 1 {
		t.Fatalf("revoked sessions = %d, want 1", revoked)
	}
	if current.RevokedAt != nil {
		t.Fatal("current session was revoked")
	}
	if other.RevokedAt == nil {
		t.Fatal("other session was not revoked")
	}
	if foreign.RevokedAt != nil {
		t.Fatal("foreign session was revoked")
	}
}

func TestRevokeOtherSessionsRejectsForeignCurrentToken(t *testing.T) {
	userID := uuid.New()
	foreign := newRefreshToken(uuid.New(), "foreign")
	repo := &refreshTokenRepositoryFake{tokens: map[uuid.UUID]*domain.RefreshToken{foreign.ID: foreign}}
	useCase := NewRefreshTokenUseCase(repo, immediateTransactionManager{}, time.Hour)

	_, err := useCase.RevokeOtherSessions(context.Background(), userID, "foreign")
	requireAppErrorType(t, err, shared.ErrTypeUnauthorized)
	if foreign.RevokedAt != nil {
		t.Fatal("foreign token should not be changed")
	}
}

func TestRevokeSessionRequiresOwnershipAndActivity(t *testing.T) {
	userID := uuid.New()
	token := newRefreshToken(userID, "current")
	foreign := newRefreshToken(uuid.New(), "foreign")
	repo := &refreshTokenRepositoryFake{tokens: map[uuid.UUID]*domain.RefreshToken{
		token.ID:   token,
		foreign.ID: foreign,
	}}
	useCase := NewRefreshTokenUseCase(repo, immediateTransactionManager{}, time.Hour)

	requireAppErrorType(t, useCase.RevokeSession(context.Background(), userID, foreign.ID), shared.ErrTypeNotFound)
	if foreign.RevokedAt != nil {
		t.Fatal("foreign session should not be changed")
	}

	if err := useCase.RevokeSession(context.Background(), userID, token.ID); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if token.RevokedAt == nil {
		t.Fatal("session was not revoked")
	}
	requireAppErrorType(t, useCase.RevokeSession(context.Background(), userID, token.ID), shared.ErrTypeNotFound)
}

func TestChangePasswordRevokesAllTokens(t *testing.T) {
	userID := uuid.New()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}
	userRepo := &userRepositoryFake{users: map[uuid.UUID]*domain.User{
		userID: {
			BaseSoftDeleteModel: shared.BaseSoftDeleteModel{BaseModel: shared.BaseModel{ID: userID}},
			Email:               "user@example.com",
			Password:            string(passwordHash),
		},
	}}
	refreshToken := newRefreshToken(userID, "refresh")
	refreshRepo := &refreshTokenRepositoryFake{tokens: map[uuid.UUID]*domain.RefreshToken{refreshToken.ID: refreshToken}}
	blacklist := &tokenBlacklistRepositoryFake{}
	refreshUseCase := NewRefreshTokenUseCase(refreshRepo, immediateTransactionManager{}, time.Hour)
	authUseCase := NewAuthUseCase(userRepo, nil, refreshUseCase, blacklist, immediateTransactionManager{}, "secret", "issuer", "audience", time.Hour)

	if err := authUseCase.ChangePassword(context.Background(), userID, "old-password", "new-password"); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(userRepo.users[userID].Password), []byte("new-password")); err != nil {
		t.Fatalf("new password does not match stored hash: %v", err)
	}
	if refreshToken.RevokedAt == nil {
		t.Fatal("refresh token was not revoked")
	}
	if len(blacklist.revokedUsers) != 1 || blacklist.revokedUsers[0] != userID.String() {
		t.Fatalf("blacklisted users = %v, want %s", blacklist.revokedUsers, userID)
	}
}

func TestChangePasswordRejectsInvalidCurrentPassword(t *testing.T) {
	userID := uuid.New()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}
	userRepo := &userRepositoryFake{users: map[uuid.UUID]*domain.User{
		userID: {
			BaseSoftDeleteModel: shared.BaseSoftDeleteModel{BaseModel: shared.BaseModel{ID: userID}},
			Password:            string(passwordHash),
		},
	}}
	blacklist := &tokenBlacklistRepositoryFake{}
	refreshUseCase := NewRefreshTokenUseCase(&refreshTokenRepositoryFake{tokens: map[uuid.UUID]*domain.RefreshToken{}}, immediateTransactionManager{}, time.Hour)
	authUseCase := NewAuthUseCase(userRepo, nil, refreshUseCase, blacklist, immediateTransactionManager{}, "secret", "issuer", "audience", time.Hour)

	err = authUseCase.ChangePassword(context.Background(), userID, "wrong-password", "new-password")
	requireAppErrorType(t, err, shared.ErrTypeUnauthorized)
	if len(blacklist.revokedUsers) != 0 {
		t.Fatalf("blacklisted users = %v, want none", blacklist.revokedUsers)
	}
}

func newRefreshToken(userID uuid.UUID, plainToken string) *domain.RefreshToken {
	return &domain.RefreshToken{
		BaseModel: shared.BaseModel{ID: uuid.New()},
		UserID:    userID,
		TokenHash: security.HashSHA256(plainToken),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
}

func requireAppErrorType(t *testing.T, err error, want shared.ErrorType) {
	t.Helper()
	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want AppError(%s)", err, want)
	}
	if appErr.Type != want {
		t.Fatalf("error type = %s, want %s", appErr.Type, want)
	}
}
