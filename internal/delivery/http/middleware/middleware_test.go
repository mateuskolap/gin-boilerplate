package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"gin-boilerplate/internal/delivery/http/authcookie"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"

	"github.com/gin-gonic/gin"
)

type middlewareAuthUseCase struct {
	domain.AuthUseCase
	claims *domain.TokenClaims
	err    error
	token  string
}

func (f *middlewareAuthUseCase) ValidateAccessToken(_ context.Context, token string) (*domain.TokenClaims, error) {
	f.token = token
	return f.claims, f.err
}

type middlewarePermissionChecker struct {
	domain.PermissionCheckerUseCase
	allowed bool
	err     error
	userID  uuid.UUID
	name    domain.PermissionName
	calls   int
}

func (f *middlewarePermissionChecker) HasPermission(_ context.Context, userID uuid.UUID, permission domain.PermissionName) (bool, error) {
	f.calls++
	f.userID, f.name = userID, permission
	return f.allowed, f.err
}

type middlewareRateLimiter struct {
	result *port.RateLimitResult
	err    error
	key    string
	limit  int
	window time.Duration
}

func (f *middlewareRateLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (*port.RateLimitResult, error) {
	f.key, f.limit, f.window = key, limit, window
	return f.result, f.err
}

func serveMiddleware(t *testing.T, method, route, target string, headers map[string]string, chain ...gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler())
	router.Handle(method, route, chain...)
	req := httptest.NewRequest(method, target, strings.NewReader(""))
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestAuthenticationMiddlewareReadsBearerAndCookieTokens(t *testing.T) {
	userID := uuid.New()
	for _, tc := range []struct {
		name       string
		useCookies bool
		headers    map[string]string
		setup      gin.HandlerFunc
	}{
		{name: "bearer", headers: map[string]string{"Authorization": "Bearer body-token"}},
		{name: "cookie", useCookies: true, setup: func(c *gin.Context) {
			c.Request.AddCookie(&http.Cookie{Name: authcookie.AccessTokenName, Value: "cookie-token"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &middlewareAuthUseCase{claims: &domain.TokenClaims{Subject: userID.String(), TokenID: "jti"}}
			nextCalled := false
			next := func(c *gin.Context) {
				gotUser, _ := c.Get("user_id")
				gotToken, _ := c.Get("raw_token")
				wantToken := "body-token"
				if tc.useCookies {
					wantToken = "cookie-token"
				}
				if gotUser != userID.String() || gotToken != wantToken {
					t.Errorf("authentication context user=%v token=%v", gotUser, gotToken)
				}
				nextCalled = true
				c.Status(http.StatusNoContent)
			}
			chain := []gin.HandlerFunc{AuthenticationMiddleware(fake, tc.useCookies)}
			if tc.setup != nil {
				chain = append([]gin.HandlerFunc{tc.setup}, chain...)
			}
			chain = append(chain, next)
			recorder := serveMiddleware(t, http.MethodGet, "/private", "/private", tc.headers, chain...)
			if recorder.Code != http.StatusNoContent || !nextCalled || fake.token == "" {
				t.Fatalf("authentication status=%d next=%v token=%q", recorder.Code, nextCalled, fake.token)
			}
		})
	}
}

func TestAuthenticationMiddlewareRejectsMissingAndInvalidTokens(t *testing.T) {
	for _, tc := range []struct {
		name    string
		header  string
		claims  *domain.TokenClaims
		err     error
		wantErr string
	}{
		{name: "missing header", wantErr: "Authorization header is required"},
		{name: "wrong scheme", header: "Basic abc", wantErr: "Authorization format must be Bearer <token>"},
		{name: "empty bearer", header: "Bearer ", wantErr: "Access token is required"},
		{name: "invalid token", header: "Bearer bad", err: shared.NewAppError(shared.ErrTypeUnauthorized, "invalid token", nil), wantErr: "invalid token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &middlewareAuthUseCase{claims: tc.claims, err: tc.err}
			called := false
			handler := func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) }
			headers := map[string]string{}
			if tc.header != "" {
				headers["Authorization"] = tc.header
			}
			recorder := serveMiddleware(t, http.MethodGet, "/private", "/private", headers, AuthenticationMiddleware(fake, false), handler)
			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusUnauthorized || called || body.Error != tc.wantErr {
				t.Fatalf("status=%d called=%v body=%s", recorder.Code, called, recorder.Body.String())
			}
		})
	}
}

func TestRequirePermissionChecksAuthenticatedUser(t *testing.T) {
	userID := uuid.New()
	for _, tc := range []struct {
		name       string
		setup      gin.HandlerFunc
		allowed    bool
		checkerErr error
		wantStatus int
	}{
		{name: "allowed", setup: func(c *gin.Context) { c.Set("user_id", userID.String()) }, allowed: true, wantStatus: http.StatusNoContent},
		{name: "denied", setup: func(c *gin.Context) { c.Set("user_id", userID.String()) }, wantStatus: http.StatusForbidden},
		{name: "checker unavailable", setup: func(c *gin.Context) { c.Set("user_id", userID.String()) }, checkerErr: errors.New("redis unavailable"), wantStatus: http.StatusInternalServerError},
		{name: "missing context", wantStatus: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checker := &middlewarePermissionChecker{allowed: tc.allowed, err: tc.checkerErr}
			called := false
			chain := []gin.HandlerFunc{}
			if tc.setup != nil {
				chain = append(chain, tc.setup)
			}
			chain = append(chain, RequirePermission(domain.PermissionViewUser, checker), func(c *gin.Context) {
				called = true
				c.Status(http.StatusNoContent)
			})
			recorder := serveMiddleware(t, http.MethodGet, "/private", "/private", nil, chain...)
			if recorder.Code != tc.wantStatus || called != (tc.wantStatus == http.StatusNoContent) {
				t.Fatalf("status=%d next called=%v body=%s", recorder.Code, called, recorder.Body.String())
			}
			if tc.setup != nil && (checker.userID != userID || checker.name != domain.PermissionViewUser || checker.calls != 1) {
				t.Fatalf("permission checker received wrong arguments: %+v", checker)
			}
		})
	}
}

func TestRequirePermissionOrOwnerSkipsCheckForOwnerAndChecksOthers(t *testing.T) {
	ownerID, otherID := uuid.New(), uuid.New()
	checker := &middlewarePermissionChecker{}
	ownerSetup := func(c *gin.Context) { c.Set("user_id", ownerID.String()) }
	owner := serveMiddleware(t, http.MethodGet, "/users/:id/image", "/users/"+ownerID.String()+"/image", nil,
		ownerSetup,
		RequirePermissionOrOwner(domain.PermissionViewUser, checker, OwnerFromUserIDParam("id")),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)
	if owner.Code != http.StatusNoContent || checker.calls != 0 {
		t.Fatalf("owner request status=%d permission checks=%d", owner.Code, checker.calls)
	}

	checker.allowed = true
	other := serveMiddleware(t, http.MethodGet, "/users/:id/image", "/users/"+otherID.String()+"/image", nil,
		ownerSetup,
		RequirePermissionOrOwner(domain.PermissionViewUser, checker, OwnerFromUserIDParam("id")),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)
	if other.Code != http.StatusNoContent || checker.calls != 1 || checker.userID != ownerID {
		t.Fatalf("permitted non-owner status=%d permission checks=%d checker=%+v", other.Code, checker.calls, checker)
	}

	checker.allowed = false
	denied := serveMiddleware(t, http.MethodGet, "/users/:id/image", "/users/"+otherID.String()+"/image", nil,
		ownerSetup,
		RequirePermissionOrOwner(domain.PermissionViewUser, checker, OwnerFromUserIDParam("id")),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("unauthorized non-owner status=%d body=%s", denied.Code, denied.Body.String())
	}
}

func TestRateLimiterHeadersAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name       string
		result     *port.RateLimitResult
		err        error
		wantStatus int
		wantRetry  string
	}{
		{name: "allowed", result: &port.RateLimitResult{Allowed: true, Limit: 10, Remaining: 9, ResetAfter: time.Minute}, wantStatus: http.StatusNoContent},
		{name: "limited", result: &port.RateLimitResult{Allowed: false, Limit: 10, Remaining: 0, RetryAfter: 1500 * time.Millisecond, ResetAfter: time.Minute}, wantStatus: http.StatusTooManyRequests, wantRetry: "2"},
		{name: "limiter unavailable", err: errors.New("redis unavailable"), wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := &middlewareRateLimiter{result: tc.result, err: tc.err}
			called := false
			recorder := serveMiddleware(t, http.MethodGet, "/things", "/things", nil,
				RateLimiter(limiter, 10, time.Minute),
				func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) },
			)
			if recorder.Code != tc.wantStatus || called != (tc.wantStatus == http.StatusNoContent) {
				t.Fatalf("status=%d called=%v body=%s", recorder.Code, called, recorder.Body.String())
			}
			if limiter.limit != 10 || limiter.window != time.Minute || !strings.Contains(limiter.key, "/things") {
				t.Fatalf("limiter received wrong key or settings: %+v", limiter)
			}
			if tc.result != nil && (recorder.Header().Get("X-RateLimit-Limit") != "10" || recorder.Header().Get("X-RateLimit-Remaining") != "9" && tc.result.Allowed) {
				t.Fatalf("rate limit headers = %v", recorder.Header())
			}
			if recorder.Header().Get("Retry-After") != tc.wantRetry {
				t.Fatalf("Retry-After = %q, want %q", recorder.Header().Get("Retry-After"), tc.wantRetry)
			}
		})
	}
}

func TestRequestIDSecurityHeadersAndRecovery(t *testing.T) {
	validID := uuid.New().String()
	for _, tc := range []struct {
		name      string
		requestID string
		wantID    string
	}{
		{name: "keep valid request ID", requestID: validID, wantID: validID},
		{name: "replace invalid request ID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(RequestID(), Recovery(), SecurityHeaders())
			router.GET("/panic", func(c *gin.Context) { panic("test panic") })
			req := httptest.NewRequest(http.MethodGet, "/panic", nil)
			if tc.requestID != "" {
				req.Header.Set("X-Request-ID", tc.requestID)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			gotID := recorder.Header().Get("X-Request-ID")
			_, parseErr := uuid.Parse(gotID)
			if recorder.Code != http.StatusInternalServerError || parseErr != nil || (tc.wantID != "" && gotID != tc.wantID) {
				t.Fatalf("recovered panic status=%d request ID=%q", recorder.Code, gotID)
			}
			for header, want := range map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"Referrer-Policy":        "no-referrer",
				"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
			} {
				if got := recorder.Header().Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}
}

func TestRequestLoggerWritesRequestMetadata(t *testing.T) {
	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	defer slog.SetDefault(previousLogger)

	requestID := uuid.New().String()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), RequestLogger())
	router.GET("/logged", func(c *gin.Context) { c.Status(http.StatusAccepted) })
	req := httptest.NewRequest(http.MethodGet, "/logged", nil)
	req.Header.Set("X-Request-ID", requestID)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("request log is not JSON: %v; output=%s", err, output.String())
	}
	if recorder.Code != http.StatusAccepted || entry["msg"] != "http request" || entry["request_id"] != requestID || entry["method"] != http.MethodGet || entry["path"] != "/logged" || entry["status"] != float64(http.StatusAccepted) {
		t.Fatalf("request log entry=%v response status=%d", entry, recorder.Code)
	}
}
