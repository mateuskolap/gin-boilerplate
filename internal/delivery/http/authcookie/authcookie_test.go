package authcookie

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSetGetAndClearAuthCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	Set(ctx, "access-value", "refresh-value", 15*time.Minute, 24*time.Hour, true)
	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Set() cookies = %v", cookies)
	}
	access, refresh := findCookie(cookies, AccessTokenName), findCookie(cookies, RefreshTokenName)
	if access == nil || access.Value != "access-value" || access.Path != "/api/v1" || !access.HttpOnly || !access.Secure || access.MaxAge != int((15*time.Minute).Seconds()) {
		t.Fatalf("access cookie = %+v", access)
	}
	if refresh == nil || refresh.Value != "refresh-value" || refresh.Path != "/api/v1/auth" || !refresh.HttpOnly || !refresh.Secure || refresh.MaxAge != int((24*time.Hour).Seconds()) {
		t.Fatalf("refresh cookie = %+v", refresh)
	}
	ctx.Request.AddCookie(access)
	if got := Get(ctx, AccessTokenName); got != "access-value" {
		t.Fatalf("Get() = %q", got)
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	Clear(ctx, false)
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.MaxAge >= 0 || cookie.Value != "" || cookie.Secure {
			t.Errorf("Clear() cookie = %+v", cookie)
		}
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
