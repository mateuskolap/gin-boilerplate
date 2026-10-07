package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	activityloghttp "gin-boilerplate/internal/activity_logs/adapters/http"
	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	organizationhttp "gin-boilerplate/internal/organizations/adapters/http"
	organizationdomain "gin-boilerplate/internal/organizations/domain"
	permissionhttp "gin-boilerplate/internal/permissions/adapters/http"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
	refreshhttp "gin-boilerplate/internal/refresh_tokens/adapters/http"
	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
	rolehttp "gin-boilerplate/internal/roles/adapters/http"
	userhttp "gin-boilerplate/internal/users/adapters/http"
	userdomain "gin-boilerplate/internal/users/domain"

	"github.com/gin-gonic/gin"
)

type routerAuthUseCase struct {
	userdomain.AuthUseCase
	claims *userdomain.TokenClaims
	err    error
}

func (f *routerAuthUseCase) ValidateAccessToken(context.Context, string) (*userdomain.TokenClaims, error) {
	return f.claims, f.err
}

type routerRefreshUseCase struct {
	refreshtokendomain.RefreshTokenUseCase
	userID uuid.UUID
	token  string
	err    error
}

type routerActivityLogUseCase struct {
	activitylogdomain.ActivityLogUseCase
	params  shared.PaginationParams
	filters []shared.Filter
}

type routerOrganizationUseCase struct {
	organizationdomain.OrganizationUseCase
}

func (routerOrganizationUseCase) List(_ context.Context, params shared.PaginationParams, _ []shared.Filter) (*shared.PaginatedResult[organizationdomain.Organization], error) {
	return &shared.PaginatedResult[organizationdomain.Organization]{Page: params.Page, Limit: params.Limit}, nil
}

func (f *routerActivityLogUseCase) List(_ context.Context, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[activitylogdomain.ActivityLog], error) {
	f.params, f.filters = params, filters
	return &shared.PaginatedResult[activitylogdomain.ActivityLog]{Page: params.Page, Limit: params.Limit}, nil
}

func (f *routerRefreshUseCase) RevokeOtherSessions(_ context.Context, userID uuid.UUID, token string) error {
	f.userID, f.token = userID, token
	return f.err
}

type routerUserUseCase struct {
	userdomain.UserUseCase
	user   *userdomain.User
	err    error
	userID uuid.UUID
}

func (f *routerUserUseCase) Find(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
	f.userID = id
	return f.user, f.err
}

type routerPermissionChecker struct {
	permissiondomain.PermissionCheckerUseCase
	allowed bool
	userID  uuid.UUID
	name    permissiondomain.PermissionName
}

func (f *routerPermissionChecker) HasPermission(_ context.Context, userID uuid.UUID, permission permissiondomain.PermissionName) (bool, error) {
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
	authHandler := userhttp.NewAuthHandler(auth, false, false, time.Minute, time.Hour, nethttp.SameSiteLaxMode)
	returnRouter, err := SetupRouter(RouterConfig{
		AuthHandler:         authHandler,
		UserHandler:         userhttp.NewUserHandler(users),
		RoleHandler:         rolehttp.NewRoleHandler(nil),
		PermissionHandler:   permissionhttp.NewPermissionHandler(nil),
		ActivityLogHandler:  activityloghttp.NewActivityLogHandler(&routerActivityLogUseCase{}),
		RefreshTokenHandler: refreshhttp.NewRefreshTokenHandler(refresh, false),
		OrganizationHandler: organizationhttp.NewOrganizationHandler(routerOrganizationUseCase{}),
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
	auth := &routerAuthUseCase{claims: &userdomain.TokenClaims{Subject: userID.String(), TokenID: "test-jti"}}
	refresh := &routerRefreshUseCase{}
	users := &routerUserUseCase{user: &userdomain.User{ID: uuid.New(), Name: "Alice"}}
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
	auth := &routerAuthUseCase{claims: &userdomain.TokenClaims{Subject: userID.String()}}
	users := &routerUserUseCase{user: &userdomain.User{ID: targetID, Name: "Target"}}
	permissions := &routerPermissionChecker{}
	router := routerForTest(t, auth, &routerRefreshUseCase{}, users, permissions)

	denied := serveRouter(router, nethttp.MethodGet, "/api/v1/users/"+targetID.String(), "", "token")
	if denied.Code != nethttp.StatusForbidden || users.userID != uuid.Nil() || permissions.name != permissiondomain.PermissionViewUser || permissions.userID != userID {
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
	auth := &routerAuthUseCase{claims: &userdomain.TokenClaims{Subject: userID.String()}}
	permissions := &routerPermissionChecker{}
	router := routerForTest(t, auth, &routerRefreshUseCase{}, &routerUserUseCase{}, permissions)

	denied := serveRouter(router, nethttp.MethodGet, "/api/v1/activity-logs", "", "token")
	if denied.Code != nethttp.StatusForbidden || permissions.name != permissiondomain.PermissionViewActivityLog || permissions.userID != userID {
		t.Fatalf("activity logs denial status=%d checker=%+v", denied.Code, permissions)
	}

	permissions.allowed = true
	allowed := serveRouter(router, nethttp.MethodGet, "/api/v1/activity-logs", "", "token")
	if allowed.Code != nethttp.StatusOK {
		t.Fatalf("activity logs allowed status=%d body=%s", allowed.Code, allowed.Body.String())
	}
}

func TestSetupRouterProtectsOrganizations(t *testing.T) {
	userID := uuid.New()
	auth := &routerAuthUseCase{claims: &userdomain.TokenClaims{Subject: userID.String()}}
	permissions := &routerPermissionChecker{}
	router := routerForTest(t, auth, &routerRefreshUseCase{}, &routerUserUseCase{}, permissions)

	denied := serveRouter(router, nethttp.MethodGet, "/api/v1/organizations", "", "token")
	if denied.Code != nethttp.StatusForbidden || permissions.name != permissiondomain.PermissionViewOrganization || permissions.userID != userID {
		t.Fatalf("organizations denial status=%d checker=%+v", denied.Code, permissions)
	}

	permissions.allowed = true
	allowed := serveRouter(router, nethttp.MethodGet, "/api/v1/organizations", "", "token")
	if allowed.Code != nethttp.StatusOK {
		t.Fatalf("organizations allowed status=%d body=%s", allowed.Code, allowed.Body.String())
	}
}

func TestSetupRouterRejectsInvalidTrustedProxy(t *testing.T) {
	_, err := SetupRouter(RouterConfig{TrustedProxies: []string{"not-a-proxy"}})
	if err == nil {
		t.Fatalf("SetupRouter() error = %v, want invalid proxy configuration", err)
	}
}
