package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"gin-boilerplate/internal/delivery/http/authcookie"
	"gin-boilerplate/internal/delivery/http/middleware"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"

	"github.com/gin-gonic/gin"
)

type httpAuthUseCase struct {
	domain.AuthUseCase
	tokens            *domain.AuthTokens
	registerErr       error
	loginErr          error
	refreshErr        error
	logoutErr         error
	changePasswordErr error
	loginArgs         []string
	refreshToken      string
	logoutArgs        []string
	changedUserID     uuid.UUID
	changedPasswords  []string
	registeredUser    *domain.User
}

func (f *httpAuthUseCase) Register(_ context.Context, user *domain.User) error {
	f.registeredUser = user
	return f.registerErr
}

func (f *httpAuthUseCase) Login(_ context.Context, email, password, ip, userAgent string) (*domain.AuthTokens, error) {
	f.loginArgs = []string{email, password, ip, userAgent}
	return f.tokens, f.loginErr
}

func (f *httpAuthUseCase) Refresh(_ context.Context, token, ip, userAgent string) (*domain.AuthTokens, error) {
	f.refreshToken = token
	f.loginArgs = []string{ip, userAgent}
	return f.tokens, f.refreshErr
}

func (f *httpAuthUseCase) Logout(_ context.Context, accessToken, refreshToken string) error {
	f.logoutArgs = []string{accessToken, refreshToken}
	return f.logoutErr
}

func (f *httpAuthUseCase) ChangePassword(_ context.Context, userID uuid.UUID, current, next string) error {
	f.changedUserID = userID
	f.changedPasswords = []string{current, next}
	return f.changePasswordErr
}

type httpRefreshTokenUseCase struct {
	domain.RefreshTokenUseCase
	listResult   *shared.PaginatedResult[domain.RefreshToken]
	listErr      error
	revokeErr    error
	otherErr     error
	listedUser   uuid.UUID
	listParams   shared.PaginationParams
	listFilters  []shared.Filter
	revokedUser  uuid.UUID
	revokedID    uuid.UUID
	otherUser    uuid.UUID
	currentToken string
	listCalls    int
}

func (f *httpRefreshTokenUseCase) ListActiveByUserID(_ context.Context, userID uuid.UUID, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[domain.RefreshToken], error) {
	f.listCalls++
	f.listedUser, f.listParams, f.listFilters = userID, params, filters
	return f.listResult, f.listErr
}

func (f *httpRefreshTokenUseCase) RevokeSession(_ context.Context, userID, sessionID uuid.UUID) error {
	f.revokedUser, f.revokedID = userID, sessionID
	return f.revokeErr
}

func (f *httpRefreshTokenUseCase) RevokeOtherSessions(_ context.Context, userID uuid.UUID, currentToken string) error {
	f.otherUser, f.currentToken = userID, currentToken
	return f.otherErr
}

func serveHTTPHandler(t *testing.T, method, route, target, body string, handler gin.HandlerFunc, setup ...func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	router.Handle(method, route, func(c *gin.Context) {
		for _, fn := range setup {
			if fn != nil {
				fn(c)
			}
		}
		handler(c)
	})
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "handler-test")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func withAuthenticatedUser(userID uuid.UUID) func(*gin.Context) {
	return func(c *gin.Context) { c.Set("user_id", userID.String()) }
}

func withRawToken(token string) func(*gin.Context) {
	return func(c *gin.Context) { c.Set("raw_token", token) }
}

func withRequestCookie(name, value string) func(*gin.Context) {
	return func(c *gin.Context) { c.Request.AddCookie(&http.Cookie{Name: name, Value: value}) }
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.ApiResponse {
	t.Helper()
	var result response.ApiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("response JSON: %v; body=%s", err, recorder.Body.String())
	}
	return result
}

func TestAuthLoginReturnsTokensInConfiguredTransport(t *testing.T) {
	for _, tc := range []struct {
		name       string
		useCookies bool
	}{
		{name: "body"},
		{name: "cookies", useCookies: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &httpAuthUseCase{tokens: &domain.AuthTokens{AccessToken: "access-value", RefreshToken: "refresh-value"}}
			handler := NewAuthHandler(fake, tc.useCookies, true, 15*time.Minute, 24*time.Hour)
			recorder := serveHTTPHandler(t, http.MethodPost, "/login", "/login", `{"email":"a@example.com","password":"pass-word"}`, handler.Login)
			if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("Login() status=%d cache-control=%q body=%s", recorder.Code, recorder.Header().Get("Cache-Control"), recorder.Body.String())
			}
			if len(fake.loginArgs) != 4 || fake.loginArgs[0] != "a@example.com" || fake.loginArgs[1] != "pass-word" || fake.loginArgs[3] != "handler-test" {
				t.Fatalf("Login() forwarded wrong credentials or client metadata: %v", fake.loginArgs)
			}
			result := decodeResponse(t, recorder)
			if !result.Success {
				t.Fatalf("Login() response = %+v", result)
			}
			cookies := recorder.Result().Cookies()
			if tc.useCookies {
				if result.Data != nil || len(cookies) != 2 {
					t.Fatalf("cookie mode returned body tokens or wrong cookies: response=%+v cookies=%v", result, cookies)
				}
				for _, name := range []string{authcookie.AccessTokenName, authcookie.RefreshTokenName} {
					cookie := findCookie(cookies, name)
					if cookie == nil || !cookie.HttpOnly || !cookie.Secure || cookie.Value == "" || cookie.SameSite != http.SameSiteLaxMode {
						t.Errorf("cookie %q has wrong attributes: %+v", name, cookie)
					}
				}
				return
			}
			data, ok := result.Data.(map[string]any)
			if !ok || data["access_token"] != "access-value" || data["refresh_token"] != "refresh-value" || len(cookies) != 0 {
				t.Fatalf("body mode response data=%v cookies=%v", result.Data, cookies)
			}
		})
	}
}

func TestAuthRegisterCreatesUserAndReturnsCreated(t *testing.T) {
	fake := &httpAuthUseCase{}
	handler := NewAuthHandler(fake, false, false, time.Minute, time.Hour)
	recorder := serveHTTPHandler(t, http.MethodPost, "/register", "/register", `{"name":"Alice","email":"alice@example.com","password":"password-123"}`, handler.Register)
	result := decodeResponse(t, recorder)
	if recorder.Code != http.StatusCreated || !result.Success || fake.registeredUser == nil || fake.registeredUser.Name != "Alice" || fake.registeredUser.Email != "alice@example.com" || fake.registeredUser.Password != "password-123" {
		t.Fatalf("Register() status=%d user=%+v response=%+v", recorder.Code, fake.registeredUser, result)
	}
}

func TestAuthRefreshReadsTokenFromConfiguredTransport(t *testing.T) {
	for _, tc := range []struct {
		name       string
		useCookies bool
		body       string
		setup      func(*gin.Context)
	}{
		{name: "body", body: `{"refresh_token":"old-body-token"}`},
		{name: "cookie", useCookies: true, setup: withRequestCookie(authcookie.RefreshTokenName, "old-cookie-token")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &httpAuthUseCase{tokens: &domain.AuthTokens{AccessToken: "new-access", RefreshToken: "new-refresh"}}
			handler := NewAuthHandler(fake, tc.useCookies, false, time.Minute, time.Hour)
			recorder := serveHTTPHandler(t, http.MethodPost, "/refresh", "/refresh", tc.body, handler.Refresh, tc.setup)
			if recorder.Code != http.StatusOK {
				t.Fatalf("Refresh() status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			want := "old-body-token"
			if tc.useCookies {
				want = "old-cookie-token"
			}
			if fake.refreshToken != want {
				t.Fatalf("Refresh() token = %q, want %q", fake.refreshToken, want)
			}
		})
	}
}

func TestAuthLogoutAcceptsBodyAndClearsCookies(t *testing.T) {
	fake := &httpAuthUseCase{}
	bodyHandler := NewAuthHandler(fake, false, false, time.Minute, time.Hour)
	bodyRecorder := serveHTTPHandler(t, http.MethodPost, "/logout", "/logout", `{"refresh_token":"refresh-body"}`, bodyHandler.Logout, withRawToken("access-from-auth-middleware"))
	if bodyRecorder.Code != http.StatusOK || len(fake.logoutArgs) != 2 || fake.logoutArgs[0] != "access-from-auth-middleware" || fake.logoutArgs[1] != "refresh-body" {
		t.Fatalf("body logout status=%d args=%v body=%s", bodyRecorder.Code, fake.logoutArgs, bodyRecorder.Body.String())
	}

	fake.logoutArgs = nil
	cookieHandler := NewAuthHandler(fake, true, true, time.Minute, time.Hour)
	cookieRecorder := serveHTTPHandler(t, http.MethodPost, "/logout", "/logout", "", cookieHandler.Logout, withRawToken("access-cookie"), withRequestCookie(authcookie.RefreshTokenName, "refresh-cookie"))
	if cookieRecorder.Code != http.StatusOK || len(fake.logoutArgs) != 2 || fake.logoutArgs[0] != "access-cookie" || fake.logoutArgs[1] != "refresh-cookie" {
		t.Fatalf("cookie logout status=%d args=%v body=%s", cookieRecorder.Code, fake.logoutArgs, cookieRecorder.Body.String())
	}
	for _, name := range []string{authcookie.AccessTokenName, authcookie.RefreshTokenName} {
		cookie := findCookie(cookieRecorder.Result().Cookies(), name)
		if cookie == nil || cookie.MaxAge >= 0 || !cookie.HttpOnly || !cookie.Secure {
			t.Errorf("logout did not clear cookie %q: %+v", name, cookie)
		}
	}
}

func TestChangePasswordReturnsNoContentAndClearsCookies(t *testing.T) {
	userID := uuid.New()
	fake := &httpAuthUseCase{}
	handler := NewAuthHandler(fake, true, true, time.Minute, time.Hour)
	recorder := serveHTTPHandler(t, http.MethodPatch, "/password", "/password", `{"current_password":"old-password","new_password":"new-password"}`, handler.ChangePassword, withAuthenticatedUser(userID))
	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 || fake.changedUserID != userID || len(fake.changedPasswords) != 2 {
		t.Fatalf("ChangePassword() status=%d body=%q user=%v passwords=%v", recorder.Code, recorder.Body.String(), fake.changedUserID, fake.changedPasswords)
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 2 || cookies[0].MaxAge >= 0 || cookies[1].MaxAge >= 0 {
		t.Fatalf("ChangePassword() did not clear auth cookies: %v", cookies)
	}
}

func TestAuthHandlerMapsInvalidPayloadAndUseCaseErrors(t *testing.T) {
	fake := &httpAuthUseCase{loginErr: shared.NewAppError(shared.ErrTypeUnauthorized, "invalid credentials", nil)}
	handler := NewAuthHandler(fake, false, false, time.Minute, time.Hour)

	badPayload := serveHTTPHandler(t, http.MethodPost, "/login", "/login", `{`, handler.Login)
	if badPayload.Code != http.StatusUnprocessableEntity {
		t.Fatalf("malformed login status = %d, body=%s", badPayload.Code, badPayload.Body.String())
	}
	unauthorized := serveHTTPHandler(t, http.MethodPost, "/login", "/login", `{"email":"a@example.com","password":"password"}`, handler.Login)
	if unauthorized.Code != http.StatusUnauthorized || decodeResponse(t, unauthorized).Error != "invalid credentials" {
		t.Fatalf("login error status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
}

func TestAuthMutationHandlersPropagateUseCaseErrors(t *testing.T) {
	userID := uuid.New()
	conflict := shared.NewAppError(shared.ErrTypeConflict, "request conflicts with existing data", nil)
	unauthorized := shared.NewAppError(shared.ErrTypeUnauthorized, "credentials are invalid", nil)

	registerFake := &httpAuthUseCase{registerErr: conflict}
	register := serveHTTPHandler(t, http.MethodPost, "/register", "/register", `{"name":"Alice","email":"alice@example.com","password":"password-123"}`, NewAuthHandler(registerFake, false, false, time.Minute, time.Hour).Register)
	if register.Code != http.StatusConflict || decodeResponse(t, register).Error != "request conflicts with existing data" {
		t.Fatalf("Register() status=%d body=%s", register.Code, register.Body.String())
	}

	refreshFake := &httpAuthUseCase{refreshErr: unauthorized}
	refresh := serveHTTPHandler(t, http.MethodPost, "/refresh", "/refresh", `{"refresh_token":"old-token"}`, NewAuthHandler(refreshFake, false, false, time.Minute, time.Hour).Refresh)
	if refresh.Code != http.StatusUnauthorized || decodeResponse(t, refresh).Error != "credentials are invalid" {
		t.Fatalf("Refresh() status=%d body=%s", refresh.Code, refresh.Body.String())
	}

	logoutFake := &httpAuthUseCase{logoutErr: unauthorized}
	logout := serveHTTPHandler(t, http.MethodPost, "/logout", "/logout", `{"refresh_token":"old-token"}`, NewAuthHandler(logoutFake, false, false, time.Minute, time.Hour).Logout, withRawToken("access-token"))
	if logout.Code != http.StatusUnauthorized || decodeResponse(t, logout).Error != "credentials are invalid" {
		t.Fatalf("Logout() status=%d body=%s", logout.Code, logout.Body.String())
	}

	passwordFake := &httpAuthUseCase{changePasswordErr: unauthorized}
	password := serveHTTPHandler(t, http.MethodPatch, "/password", "/password", `{"current_password":"old-password","new_password":"new-password"}`, NewAuthHandler(passwordFake, true, true, time.Minute, time.Hour).ChangePassword, withAuthenticatedUser(userID))
	if password.Code != http.StatusUnauthorized || decodeResponse(t, password).Error != "credentials are invalid" || len(password.Result().Cookies()) != 0 {
		t.Fatalf("ChangePassword() status=%d body=%s cookies=%v", password.Code, password.Body.String(), password.Result().Cookies())
	}
}

func TestRefreshTokenHandlersUseAuthenticatedOwnerAndDELETEBody(t *testing.T) {
	userID, sessionID := uuid.New(), uuid.New()
	fake := &httpRefreshTokenUseCase{listResult: &shared.PaginatedResult[domain.RefreshToken]{Page: 2, Limit: 5, Total: 7}}
	handler := NewRefreshTokenHandler(fake, false)
	setupUser := withAuthenticatedUser(userID)

	list := serveHTTPHandler(t, http.MethodGet, "/sessions", "/sessions?page=2&limit=5&sort=-created_at&user_agent=Browser&ip_address=192.0.2", "", handler.ListRefreshTokensByAuthUser, setupUser)
	if list.Code != http.StatusOK || fake.listedUser != userID || fake.listParams.Page != 2 || fake.listParams.Limit != 5 || len(fake.listParams.Sort) != 1 || fake.listParams.Sort[0].Direction != shared.SortDesc || len(fake.listFilters) != 2 {
		t.Fatalf("session list status=%d user=%v params=%+v filters=%+v", list.Code, fake.listedUser, fake.listParams, fake.listFilters)
	}
	if fake.listFilters[0].Value != "%Browser%" || fake.listFilters[1].Value != "%192.0.2%" {
		t.Fatalf("session list filters = %+v", fake.listFilters)
	}

	revoke := serveHTTPHandler(t, http.MethodDelete, "/sessions/:id", "/sessions/"+sessionID.String(), "", handler.RevokeSession, setupUser)
	if revoke.Code != http.StatusNoContent || revoke.Body.Len() != 0 || fake.revokedUser != userID || fake.revokedID != sessionID {
		t.Fatalf("RevokeSession() status=%d body=%q owner=%v id=%v", revoke.Code, revoke.Body.String(), fake.revokedUser, fake.revokedID)
	}

	body := `{"current_refresh_token":"keep-this-session"}`
	others := serveHTTPHandler(t, http.MethodDelete, "/sessions/revoke-others", "/sessions/revoke-others", body, handler.RevokeOtherSessions, setupUser)
	if others.Code != http.StatusNoContent || others.Body.Len() != 0 || fake.otherUser != userID || fake.currentToken != "keep-this-session" {
		t.Fatalf("RevokeOtherSessions() status=%d body=%q owner=%v current=%q", others.Code, others.Body.String(), fake.otherUser, fake.currentToken)
	}
}

func TestRevokeOtherSessionsUsesCookieAndRejectsInvalidBody(t *testing.T) {
	userID := uuid.New()
	fake := &httpRefreshTokenUseCase{}
	cookieHandler := NewRefreshTokenHandler(fake, true)
	cookieResponse := serveHTTPHandler(t, http.MethodDelete, "/sessions/revoke-others", "/sessions/revoke-others", "", cookieHandler.RevokeOtherSessions, withAuthenticatedUser(userID), withRequestCookie(authcookie.RefreshTokenName, "cookie-current"))
	if cookieResponse.Code != http.StatusNoContent || fake.currentToken != "cookie-current" {
		t.Fatalf("cookie revoke others status=%d token=%q", cookieResponse.Code, fake.currentToken)
	}

	bodyHandler := NewRefreshTokenHandler(fake, false)
	badBody := serveHTTPHandler(t, http.MethodDelete, "/sessions/revoke-others", "/sessions/revoke-others", `{"current_refresh_token":""}`, bodyHandler.RevokeOtherSessions, withAuthenticatedUser(userID))
	if badBody.Code != http.StatusUnprocessableEntity || fake.currentToken != "cookie-current" {
		t.Fatalf("invalid DELETE body status=%d token=%q body=%s", badBody.Code, fake.currentToken, badBody.Body.String())
	}
}

func TestSessionHandlerErrorsAndMissingAuthentication(t *testing.T) {
	userID := uuid.New()
	fake := &httpRefreshTokenUseCase{revokeErr: shared.NewAppError(shared.ErrTypeNotFound, "session not found", nil)}
	handler := NewRefreshTokenHandler(fake, false)
	missingUser := serveHTTPHandler(t, http.MethodDelete, "/sessions/:id", "/sessions/"+uuid.New().String(), "", handler.RevokeSession)
	if missingUser.Code != http.StatusUnauthorized {
		t.Fatalf("missing user status=%d body=%s", missingUser.Code, missingUser.Body.String())
	}
	invalidID := serveHTTPHandler(t, http.MethodDelete, "/sessions/:id", "/sessions/not-a-uuid", "", handler.RevokeSession, withAuthenticatedUser(userID))
	if invalidID.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid session ID status=%d body=%s", invalidID.Code, invalidID.Body.String())
	}
	notFound := serveHTTPHandler(t, http.MethodDelete, "/sessions/:id", "/sessions/"+uuid.New().String(), "", handler.RevokeSession, withAuthenticatedUser(userID))
	if notFound.Code != http.StatusNotFound || decodeResponse(t, notFound).Error != "session not found" {
		t.Fatalf("missing session status=%d body=%s", notFound.Code, notFound.Body.String())
	}
}

func TestSessionListAndRevokeOthersPropagateUseCaseErrors(t *testing.T) {
	userID := uuid.New()
	notFound := shared.NewAppError(shared.ErrTypeNotFound, "session not found", nil)
	listFake := &httpRefreshTokenUseCase{listErr: notFound}
	list := serveHTTPHandler(t, http.MethodGet, "/sessions", "/sessions", "", NewRefreshTokenHandler(listFake, false).ListRefreshTokensByAuthUser, withAuthenticatedUser(userID))
	if list.Code != http.StatusNotFound || decodeResponse(t, list).Error != "session not found" {
		t.Fatalf("session list status=%d body=%s", list.Code, list.Body.String())
	}

	otherFake := &httpRefreshTokenUseCase{otherErr: notFound}
	others := serveHTTPHandler(t, http.MethodDelete, "/sessions/revoke-others", "/sessions/revoke-others", `{"current_refresh_token":"current-token"}`, NewRefreshTokenHandler(otherFake, false).RevokeOtherSessions, withAuthenticatedUser(userID))
	if others.Code != http.StatusNotFound || decodeResponse(t, others).Error != "session not found" {
		t.Fatalf("revoke other sessions status=%d body=%s", others.Code, others.Body.String())
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
