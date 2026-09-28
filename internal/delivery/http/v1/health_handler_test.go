package v1

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type healthChecker struct{ err error }

func (h healthChecker) Readiness(context.Context) error { return h.err }

func TestHealthHandlerLivenessAndReadiness(t *testing.T) {
	live := serveHTTPHandler(t, http.MethodGet, "/health/live", "/health/live", "", NewHealthHandler(healthChecker{}).Liveness)
	if live.Code != http.StatusOK {
		t.Fatalf("Liveness() status=%d body=%s", live.Code, live.Body.String())
	}
	ready := serveHTTPHandler(t, http.MethodGet, "/health/ready", "/health/ready", "", NewHealthHandler(healthChecker{}).Readiness)
	if ready.Code != http.StatusOK {
		t.Fatalf("Readiness() status=%d body=%s", ready.Code, ready.Body.String())
	}
	notReady := serveHTTPHandler(t, http.MethodGet, "/health/ready", "/health/ready", "", NewHealthHandler(healthChecker{err: errors.New("database unavailable")}).Readiness)
	if notReady.Code != http.StatusServiceUnavailable {
		t.Fatalf("Readiness() on dependency error status=%d body=%s", notReady.Code, notReady.Body.String())
	}
}
