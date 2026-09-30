package application_test

import (
	"context"
	"errors"
	"time"
	"uuid"

	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	"gin-boilerplate/internal/infra/security"
	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
	roledomain "gin-boilerplate/internal/roles/domain"
	userapp "gin-boilerplate/internal/users/application"
	userdomain "gin-boilerplate/internal/users/domain"
)

type testUserRepo struct {
	userdomain.UserRepository
	byEmail       *userdomain.User
	byID          map[uuid.UUID]*userdomain.User
	getByEmailErr error
	getByIDErr    error
	createErr     error
	updateErr     error
	lastEmail     string
}

func (r *testUserRepo) GetByEmail(_ context.Context, email string) (*userdomain.User, error) {
	r.lastEmail = email
	if r.getByEmailErr != nil {
		return nil, r.getByEmailErr
	}
	if r.byEmail != nil && r.byEmail.Email == email {
		return r.byEmail, nil
	}
	return nil, nil
}

func (r *testUserRepo) GetByID(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	return r.byID[id], nil
}

func (r *testUserRepo) GetByIDWithRoles(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
	return r.GetByID(ctx, id)
}

func (r *testUserRepo) Create(_ context.Context, user *userdomain.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	if user.ID == uuid.Nil() {
		user.ID = uuid.New()
	}
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*userdomain.User)
	}
	r.byID[user.ID] = user
	r.byEmail = user
	return nil
}

func (r *testUserRepo) Update(_ context.Context, user *userdomain.User) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*userdomain.User)
	}
	r.byID[user.ID] = user
	return nil
}

type testRoleRepo struct {
	roledomain.RoleRepository
	byName *roledomain.Role
	err    error
}

func (r *testRoleRepo) GetByName(_ context.Context, _ string) (*roledomain.Role, error) {
	return r.byName, r.err
}

type testRefreshUseCase struct {
	refreshtokendomain.RefreshTokenUseCase
	createdToken     string
	createErr        error
	validatedToken   *refreshtokendomain.RefreshToken
	validatedInput   string
	rotatedToken     string
	validateErr      error
	rotateErr        error
	revokeErr        error
	revokeAllErr     error
	createdUserID    uuid.UUID
	createdIP        string
	createdUserAgent string
	rotatedOldToken  string
	rotatedIP        string
	rotatedUserAgent string
	revokedToken     string
	revokedAllUserID uuid.UUID
}

func (r *testRefreshUseCase) Create(_ context.Context, userID uuid.UUID, ip, userAgent string) (string, error) {
	r.createdUserID, r.createdIP, r.createdUserAgent = userID, ip, userAgent
	return r.createdToken, r.createErr
}

func (r *testRefreshUseCase) Validate(_ context.Context, token string) (*refreshtokendomain.RefreshToken, error) {
	r.validatedInput = token
	return r.validatedToken, r.validateErr
}

func (r *testRefreshUseCase) Rotate(_ context.Context, oldToken, ip, userAgent string) (string, error) {
	r.rotatedOldToken, r.rotatedIP, r.rotatedUserAgent = oldToken, ip, userAgent
	return r.rotatedToken, r.rotateErr
}

func (r *testRefreshUseCase) Revoke(_ context.Context, token string) error {
	r.revokedToken = token
	return r.revokeErr
}

func (r *testRefreshUseCase) RevokeAllByUserID(_ context.Context, userID uuid.UUID) error {
	r.revokedAllUserID = userID
	return r.revokeAllErr
}

type testBlacklist struct {
	revokedTokenID  string
	revokedTokenTTL time.Duration
	revokeTokenErr  error
	revokedUserID   string
	revokeUserTTL   time.Duration
	revokeUserErr   error
	tokenIsRevoked  bool
	userIsRevoked   bool
	checkTokenErr   error
	checkUserErr    error
	checkedTokenID  string
	checkedUserID   string
	checkedIssuedAt time.Time
}

func (b *testBlacklist) RevokeToken(_ context.Context, id string, ttl time.Duration) error {
	b.revokedTokenID, b.revokedTokenTTL = id, ttl
	return b.revokeTokenErr
}

func (b *testBlacklist) IsRevoked(_ context.Context, tokenID string) (bool, error) {
	b.checkedTokenID = tokenID
	return b.tokenIsRevoked, b.checkTokenErr
}

func (b *testBlacklist) RevokeUserTokens(_ context.Context, userID string, ttl time.Duration) error {
	b.revokedUserID, b.revokeUserTTL = userID, ttl
	return b.revokeUserErr
}

func (b *testBlacklist) IsUserTokenRevoked(_ context.Context, userID string, issuedAt time.Time) (bool, error) {
	b.checkedUserID, b.checkedIssuedAt = userID, issuedAt
	return b.userIsRevoked, b.checkUserErr
}

type testTransaction struct {
	calls int
	err   error
}

type testActivityLogRepo struct {
	activitylogdomain.ActivityLogRepository
	activities []*activitylogdomain.ActivityLog
	createErr  error
	listErr    error
	listResult *shared.PaginatedResult[activitylogdomain.ActivityLog]
}

func (r *testActivityLogRepo) Create(_ context.Context, activity *activitylogdomain.ActivityLog) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.activities = append(r.activities, activity)
	return nil
}

func (r *testActivityLogRepo) List(_ context.Context, _ shared.PaginationParams, _ []shared.Filter) (*shared.PaginatedResult[activitylogdomain.ActivityLog], error) {
	return r.listResult, r.listErr
}

func (tx *testTransaction) Do(ctx context.Context, fn func(context.Context) error) error {
	tx.calls++
	if err := fn(ctx); err != nil {
		return err
	}
	return tx.err
}

type testRefreshRepo struct {
	refreshtokendomain.RefreshTokenRepository
	tokens             []*refreshtokendomain.RefreshToken
	createErr          error
	findErr            error
	revokeErr          error
	revokeAllErr       error
	revokeExceptErr    error
	listErr            error
	deleteErr          error
	deleteBefore       time.Time
	deleteCount        int64
	listFilters        []shared.Filter
	listParams         shared.PaginationParams
	listUserID         uuid.UUID
	listResult         *shared.PaginatedResult[refreshtokendomain.RefreshToken]
	listCalls          int
	revokedUserIDs     []uuid.UUID
	revokeExceptUserID uuid.UUID
	revokeExceptID     uuid.UUID
}

func (r *testRefreshRepo) Create(_ context.Context, token *refreshtokendomain.RefreshToken) error {
	if r.createErr != nil {
		return r.createErr
	}
	if token.ID == uuid.Nil() {
		token.ID = uuid.New()
	}
	r.tokens = append(r.tokens, token)
	return nil
}

func (r *testRefreshRepo) FindByTokenHash(_ context.Context, hash string) (*refreshtokendomain.RefreshToken, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	for _, token := range r.tokens {
		if token.TokenHash == hash {
			return token, nil
		}
	}
	return nil, nil
}

func (r *testRefreshRepo) Revoke(_ context.Context, id, userID uuid.UUID, replacedBy *uuid.UUID) (bool, error) {
	if r.revokeErr != nil {
		return false, r.revokeErr
	}
	for _, token := range r.tokens {
		if token.ID == id && token.UserID == userID && token.RevokedAt == nil && time.Now().UTC().Before(token.ExpiresAt) {
			now := time.Now().UTC()
			token.RevokedAt = &now
			token.ReplacedBy = replacedBy
			return true, nil
		}
	}
	return false, nil
}

func (r *testRefreshRepo) RevokeAllByUserID(_ context.Context, userID uuid.UUID) error {
	r.revokedUserIDs = append(r.revokedUserIDs, userID)
	return r.revokeAllErr
}

func (r *testRefreshRepo) RevokeAllExcept(_ context.Context, userID, exceptID uuid.UUID) error {
	r.revokeExceptUserID, r.revokeExceptID = userID, exceptID
	return r.revokeExceptErr
}

func (r *testRefreshRepo) ListByUserID(_ context.Context, userID uuid.UUID, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[refreshtokendomain.RefreshToken], error) {
	r.listCalls++
	r.listUserID, r.listParams = userID, params
	r.listFilters = filters
	return r.listResult, r.listErr
}

func (r *testRefreshRepo) DeleteExpiredBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.deleteBefore = cutoff
	return r.deleteCount, r.deleteErr
}

func newTestAuthUseCase() (userdomain.AuthUseCase, *testUserRepo, *testRoleRepo, *testRefreshUseCase, *testBlacklist, *testTransaction) {
	auth, userRepo, roleRepo, refresh, blacklist, tx, _ := newTestAuthUseCaseWithPasswordLevel(1)
	return auth, userRepo, roleRepo, refresh, blacklist, tx
}

func newTestAuthUseCaseWithPasswordLevel(level int) (userdomain.AuthUseCase, *testUserRepo, *testRoleRepo, *testRefreshUseCase, *testBlacklist, *testTransaction, *testPasswordChecker) {
	userRepo := &testUserRepo{byID: make(map[uuid.UUID]*userdomain.User)}
	roleRepo := &testRoleRepo{}
	refresh := &testRefreshUseCase{}
	blacklist := &testBlacklist{}
	tx := &testTransaction{}
	checker := &testPasswordChecker{}
	auth := userapp.NewAuthUseCase(userRepo, roleRepo, refresh, blacklist, tx, "test-secret-with-at-least-32-characters", "issuer", "audience", time.Hour, level, checker, allowAllRateLimiter{})
	return auth, userRepo, roleRepo, refresh, blacklist, tx, checker
}

type testPasswordChecker struct {
	compromised bool
	err         error
	calls       int
}

func (c *testPasswordChecker) IsCompromised(context.Context, string) (bool, error) {
	c.calls++
	return c.compromised, c.err
}

type allowAllRateLimiter struct{}

func (allowAllRateLimiter) Allow(context.Context, string, int, time.Duration) (*port.RateLimitResult, error) {
	return &port.RateLimitResult{Allowed: true}, nil
}

func appErrorType(err error) shared.ErrorType {
	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		return ""
	}
	return appErr.Type
}

func testAccessToken(userID uuid.UUID) (string, error) {
	return security.GenerateAccessToken(userID, "test-secret-with-at-least-32-characters", "issuer", "audience", time.Hour)
}
