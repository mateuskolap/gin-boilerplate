package security

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type testPasswordRoundTripper func(*http.Request) (*http.Response, error)

func (f testPasswordRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestPwnedPasswordCheckerUsesHashRangeAndMatchesLocally(t *testing.T) {
	password := "Passw0rd!"
	hash := sha1.Sum([]byte(password))
	encodedHash := strings.ToUpper(hex.EncodeToString(hash[:]))
	userAgent, err := applicationUserAgent()
	if err != nil {
		t.Fatal(err)
	}
	checker := NewPwnedPasswordChecker()
	checker.client = &http.Client{Transport: testPasswordRoundTripper(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "api.pwnedpasswords.com" || request.URL.Path != "/range/"+encodedHash[:5] || request.Header.Get("Add-Padding") != "true" || request.Header.Get("User-Agent") != userAgent {
			t.Errorf("unexpected HIBP request: %s headers=%v", request.URL, request.Header)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(encodedHash[5:] + ":12\r\n"))}, nil
	})}

	compromised, err := checker.IsCompromised(context.Background(), password)
	if err != nil || !compromised {
		t.Fatalf("IsCompromised() = (%v, %v), want (true, nil)", compromised, err)
	}
}

func TestPwnedPasswordCheckerIgnoresPaddingAndReportsFailures(t *testing.T) {
	password := "Passw0rd!"
	hash := sha1.Sum([]byte(password))
	suffix := strings.ToUpper(hex.EncodeToString(hash[:]))[5:]

	t.Run("zero-count padding", func(t *testing.T) {
		checker := NewPwnedPasswordChecker()
		checker.client = &http.Client{Transport: testPasswordRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(suffix + ":0\r\n"))}, nil
		})}
		compromised, err := checker.IsCompromised(context.Background(), password)
		if err != nil || compromised {
			t.Fatalf("IsCompromised() = (%v, %v), want (false, nil)", compromised, err)
		}
	})

	t.Run("service error", func(t *testing.T) {
		checker := NewPwnedPasswordChecker()
		checker.client = &http.Client{Transport: testPasswordRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader(""))}, nil
		})}
		if _, err := checker.IsCompromised(context.Background(), password); err == nil {
			t.Fatal("IsCompromised() returned no error for an unavailable service")
		}
	})

	t.Run("request timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		checker := NewPwnedPasswordChecker()
		checker.client = &http.Client{Transport: testPasswordRoundTripper(func(request *http.Request) (*http.Response, error) {
			<-request.Context().Done()
			return nil, request.Context().Err()
		})}
		if _, err := checker.IsCompromised(ctx, password); err == nil {
			t.Fatal("IsCompromised() returned no error after the request timed out")
		}
	})
}
