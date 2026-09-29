package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"

	"github.com/gin-gonic/gin"
)

type routerAuthUseCase struct {
	domain.AuthUseCase
	claims *domain.TokenClaims
	err    error
}

func (f *routerAuthUseCase) ValidateAccessToken(context.Context, string) (*domain.TokenClaims, error) {
	return f.claims, f.err
}

type routerRefreshUseCase struct {
	domain.RefreshTokenUseCase
	userID uuid.UUID
	token  string
	err    error
}

type routerActivityLogUseCase struct {
	domain.ActivityLogUseCase
	params  shared.PaginationParams
	filters []shared.Filter
}

func (f *routerActivityLogUseCase) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[domain.ActivityLog], error) {
	f.params, f.filters = params, filters
	return &shared.PaginatedResult[domain.ActivityLog]{Page: params.Page, Limit: params.Limit}, nil
}

func (f *routerRefreshUseCase) RevokeOtherSessions(_ context.Context, userID uuid.UUID, token string) error {
	f.userID, f.token = userID, token
	return f.err
}

type routerUserUseCase struct {
	domain.UserUseCase
	user   *domain.User
	err    error
	userID uuid.UUID
}

func (f *routerUserUseCase) Find(_ context.Context, id uuid.UUID) (*domain.User, error) {
	f.userID = id
	return f.user, f.err
}

type routerPermissionChecker struct {
	domain.PermissionCheckerUseCase
	allowed bool
	userID  uuid.UUID
	name    domain.PermissionName
}

func (f *routerPermissionChecker) HasPermission(_ context.Context, userID uuid.UUID, permission domain.PermissionName) (bool, error) {
	f.userID, f.name = userID, permission
	return f.allowed, nil
}

type routerRateLimiter struct{}

func (routerRateLimiter) Allow(context.Context, string, int, time.Duration) (*port.RateLimitResult, error) {
	return &port.RateLimitResult{Allowed: true, Limit: 10, Remaining: 9}, nil
}

type routerHealthChecker struct{}

func (routerHealthChecker) Readiness(context.Context) error { return nil }

func routerForTest(t *testing.T, auth *routerAuthUseCase, refresh *routerRefreshUseCase, users *routerUserUseCase, permissions *routerPermissionChecker) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	authHandler := v1.NewAuthHandler(auth, false, false, time.Minute, time.Hour, nethttp.SameSiteLaxMode)
	returnRouter, err := SetupRouter(RouterConfig{
		AuthHandler:         authHandler,
		UserHandler:         v1.NewUserHandler(users),
		RoleHandler:         v1.NewRoleHandler(nil),
		PermissionHandler:   v1.NewPermissionHandler(nil),
		ActivityLogHandler:  v1.NewActivityLogHandler(&routerActivityLogUseCase{}),
		RefreshTokenHandler: v1.NewRefreshTokenHandler(refresh, false),
		HealthHandler:       v1.NewHealthHandler(routerHealthChecker{}),
		AuthUseCase:         auth,
		PermissionChecker:   permissions,
		RateLimiter:         routerRateLimiter{},
		Env:                 "test",
	})
	if err != nil {
		t.Fatalf("SetupRouter() error = %v", err)
	}
	return returnRouter
}

func serveRouter(router *gin.Engine, method, target, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestSetupRouterProtectsRoutesAndWiresDELETEBody(t *testing.T) {
	userID := uuid.New()
	auth := &routerAuthUseCase{claims: &domain.TokenClaims{Subject: userID.String(), TokenID: "test-jti"}}
	refresh := &routerRefreshUseCase{}
	users := &routerUserUseCase{user: &domain.User{ID: uuid.New(), Name: "Alice"}}
	permissions := &routerPermissionChecker{}
	router := routerForTest(t, auth, refresh, users, permissions)

	unauthenticated := serveRouter(router, nethttp.MethodGet, "/api/v1/users/profile", "", "")
	if unauthenticated.Code != nethttp.StatusUnauthorized {
		t.Fatalf("protected route without token status=%d body=%s", unauthenticated.Code, unauthenticated.Body.String())
	}

	profile := serveRouter(router, nethttp.MethodGet, "/api/v1/users/profile", "", "valid-token")
	if profile.Code != nethttp.StatusOK || users.userID != userID || profile.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("authenticated profile status=%d user=%v headers=%v", profile.Code, users.userID, profile.Header())
	}

	body := `{"current_refresh_token":"keep-session"}`
	revokeOthers := serveRouter(router, nethttp.MethodDelete, "/api/v1/auth/sessions/revoke-others", body, "valid-token")
	if revokeOthers.Code != nethttp.StatusNoContent || revokeOthers.Body.Len() != 0 || refresh.userID != userID || refresh.token != "keep-session" {
		t.Fatalf("DELETE with body status=%d body=%q user=%v token=%q", revokeOthers.Code, revokeOthers.Body.String(), refresh.userID, refresh.token)
	}
	if _, err := uuid.Parse(revokeOthers.Header().Get("X-Request-ID")); err != nil {
		t.Fatalf("router did not emit a valid request ID: %q", revokeOthers.Header().Get("X-Request-ID"))
	}
}

func TestSetupRouterAppliesPermissionGuard(t *testing.T) {
	userID, targetID := uuid.New(), uuid.New()
	auth := &routerAuthUseCase{claims: &domain.TokenClaims{Subject: userID.String()}}
	users := &routerUserUseCase{user: &domain.User{ID: targetID, Name: "Target"}}
	permissions := &routerPermissionChecker{}
	router := routerForTest(t, auth, &routerRefreshUseCase{}, users, permissions)

	denied := serveRouter(router, nethttp.MethodGet, "/api/v1/users/"+targetID.String(), "", "token")
	if denied.Code != nethttp.StatusForbidden || users.userID != uuid.Nil() || permissions.name != domain.PermissionViewUser || permissions.userID != userID {
		t.Fatalf("permission denied route status=%d user lookup=%v checker=%+v", denied.Code, users.userID, permissions)
	}

	permissions.allowed = true
	allowed := serveRouter(router, nethttp.MethodGet, "/api/v1/users/"+targetID.String(), "", "token")
	if allowed.Code != nethttp.StatusOK || users.userID != targetID {
		t.Fatalf("permitted route status=%d user lookup=%v body=%s", allowed.Code, users.userID, allowed.Body.String())
	}
}

func TestSetupRouterProtectsActivityLogs(t *testing.T) {
	userID := uuid.New()
	auth := &routerAuthUseCase{claims: &domain.TokenClaims{Subject: userID.String()}}
	permissions := &routerPermissionChecker{}
	router := routerForTest(t, auth, &routerRefreshUseCase{}, &routerUserUseCase{}, permissions)

	denied := serveRouter(router, nethttp.MethodGet, "/api/v1/activity-logs", "", "token")
	if denied.Code != nethttp.StatusForbidden || permissions.name != domain.PermissionViewActivityLog || permissions.userID != userID {
		t.Fatalf("activity logs denial status=%d checker=%+v", denied.Code, permissions)
	}

	permissions.allowed = true
	allowed := serveRouter(router, nethttp.MethodGet, "/api/v1/activity-logs", "", "token")
	if allowed.Code != nethttp.StatusOK {
		t.Fatalf("activity logs allowed status=%d body=%s", allowed.Code, allowed.Body.String())
	}
}

func TestSetupRouterRejectsInvalidTrustedProxy(t *testing.T) {
	_, err := SetupRouter(RouterConfig{TrustedProxies: []string{"not-a-proxy"}})
	if err == nil {
		t.Fatalf("SetupRouter() error = %v, want invalid proxy configuration", err)
	}
}
