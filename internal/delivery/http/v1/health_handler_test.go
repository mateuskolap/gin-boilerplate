package v1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type healthChecker struct{ err error }

func (h healthChecker) Readiness(context.Context) error { return h.err }

func serveHealthHandler(t *testing.T, method, path string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path, handler)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
	return recorder
}

func TestHealthHandlerLivenessAndReadiness(t *testing.T) {
	live := serveHealthHandler(t, http.MethodGet, "/health/live", NewHealthHandler(healthChecker{}).Liveness)
	if live.Code != http.StatusOK {
		t.Fatalf("Liveness() status=%d body=%s", live.Code, live.Body.String())
	}
	ready := serveHealthHandler(t, http.MethodGet, "/health/ready", NewHealthHandler(healthChecker{}).Readiness)
	if ready.Code != http.StatusOK {
		t.Fatalf("Readiness() status=%d body=%s", ready.Code, ready.Body.String())
	}
	notReady := serveHealthHandler(t, http.MethodGet, "/health/ready", NewHealthHandler(healthChecker{err: errors.New("database unavailable")}).Readiness)
	if notReady.Code != http.StatusServiceUnavailable {
		t.Fatalf("Readiness() on dependency error status=%d body=%s", notReady.Code, notReady.Body.String())
	}
}
