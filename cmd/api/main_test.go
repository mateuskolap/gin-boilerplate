package main

import (
	"strings"
	"testing"
)

func TestRunRejectsMissingJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	if err := run(); err == nil || !strings.Contains(err.Error(), "load configuration") {
		t.Fatalf("run() error = %v, want configuration error", err)
	}
}
