package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gin-boilerplate/internal/bootstrap"
	"gin-boilerplate/internal/delivery/http/authcookie"
	"gin-boilerplate/internal/users/adapters/http/dto"

	"uuid"
)

func TestAuthenticationE2E(t *testing.T) {
	for _, transport := range []string{"body", "cookie"} {
		t.Run(transport, func(t *testing.T) {
			cfg := testApplicationConfig(t)
			db := testPostgresSchema(t)
			var schema string
			if err := db.Raw("SELECT current_schema()").Scan(&schema).Error; err != nil {
				t.Fatal(err)
			}
			// Scope the bootstrap's pgx connections to this test's isolated schema.
			t.Setenv("PGOPTIONS", "-c search_path="+schema)
			migrations, err := filepath.Glob("../../db/migrations/*.up.sql")
			if err != nil || len(migrations) == 0 {
				t.Fatalf("find SQL migrations: %v", err)
			}
			for _, migration := range migrations {
				sql, err := os.ReadFile(migration)
				if err != nil {
					t.Fatal(err)
				}
				if err := db.Exec(string(sql)).Error; err != nil {
					t.Fatalf("apply %s: %v", migration, err)
				}
			}
			cfg.AuthTokenTransport = transport
			const frontendOrigin = "https://frontend.example.com"
			if transport == "cookie" {
				cfg.AuthCookieSameSite = "none"
				cfg.CORSAllowedOrigins = []string{frontendOrigin}
			}
			cfg.PasswordValidationLevel = 1
			cfg.AdminName = "E2E Admin"
			cfg.AdminEmail = uuid.New().String() + "@example.com"
			cfg.AdminPassword = "e2e-admin-password"
			app, err := bootstrap.NewApplication(cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := app.Close(); err != nil {
					t.Errorf("close application: %v", err)
				}
			})
			if err := app.Seed(context.Background()); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewTLSServer(app.Router)
			t.Cleanup(server.Close)
			jar, err := cookiejar.New(nil)
			if err != nil {
				t.Fatal(err)
			}
			client := server.Client()
			client.Jar = jar
			client.Timeout = 5 * time.Second
			if transport == "cookie" {
				req, err := http.NewRequest(http.MethodOptions, server.URL+"/api/v1/users/profile", nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Origin", frontendOrigin)
				req.Header.Set("Access-Control-Request-Method", http.MethodPut)
				req.Header.Set("Access-Control-Request-Headers", "content-type")
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != frontendOrigin || resp.Header.Get("Access-Control-Allow-Credentials") != "true" {
					t.Fatal("CORS preflight must permit the configured frontend with credentials")
				}
			}
			authURL, err := url.Parse(server.URL + "/api/v1/auth")
			if err != nil {
				t.Fatal(err)
			}
			var responseCookies []*http.Cookie
			request := func(method, path string, payload any, accessToken string, wantStatus int) json.RawMessage {
				t.Helper()
				var body []byte
				if payload != nil {
					body, err = json.Marshal(payload)
					if err != nil {
						t.Fatal(err)
					}
				}
				req, err := http.NewRequest(method, server.URL+"/api/v1"+path, bytes.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				if payload != nil {
					req.Header.Set("Content-Type", "application/json")
				}
				if transport == "cookie" {
					req.Header.Set("Origin", frontendOrigin)
				}
				if transport == "body" && accessToken != "" {
					req.Header.Set("Authorization", "Bearer "+accessToken)
				}
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				body, err = io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				if resp.StatusCode != wantStatus {
					t.Fatalf("%s %s: status=%d, want %d; response=%s", method, path, resp.StatusCode, wantStatus, body)
				}
				var result struct {
					Success bool            `json:"success"`
					Data    json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatal(err)
				}
				if result.Success != (wantStatus < 400) {
					t.Fatalf("%s %s: unexpected success flag", method, path)
				}
				responseCookies = resp.Cookies()
				return result.Data
			}
			tokens := func(data json.RawMessage) dto.LoginResponse {
				t.Helper()
				var pair dto.LoginResponse
				if transport == "body" {
					if err := json.Unmarshal(data, &pair); err != nil {
						t.Fatal(err)
					}
					if len(responseCookies) != 0 {
						t.Fatal("body mode unexpectedly set authentication cookies")
					}
				} else {
					if len(data) != 0 {
						t.Fatal("cookie mode exposed tokens in the response body")
					}
					if len(responseCookies) != 2 || !responseCookies[0].HttpOnly || !responseCookies[1].HttpOnly {
						t.Fatal("cookie mode must set two HttpOnly cookies")
					}
					for _, cookie := range responseCookies {
						if !cookie.Secure || cookie.SameSite != http.SameSiteNoneMode {
							t.Fatal("cross-site cookie mode must use SameSite=None and Secure")
						}
					}
					for _, cookie := range jar.Cookies(authURL) {
						switch cookie.Name {
						case authcookie.AccessTokenName:
							pair.AccessToken = cookie.Value
						case authcookie.RefreshTokenName:
							pair.RefreshToken = cookie.Value
						}
					}
				}
				if pair.AccessToken == "" || pair.RefreshToken == "" {
					t.Fatal("missing authentication tokens")
				}
				return pair
			}

			email := uuid.New().String() + "@example.com"
			credentials := dto.LoginRequest{Email: email, Password: "e2e-user-password"}
			request(http.MethodGet, "/users/profile", nil, "", http.StatusUnauthorized)
			registered := request(http.MethodPost, "/auth/register", dto.RegisterRequest{
				Name: "E2E User", Email: email, Password: credentials.Password,
			}, "", http.StatusCreated)
			var user dto.UserResponse
			if err := json.Unmarshal(registered, &user); err != nil || user.ID == uuid.Nil() || user.Email != email {
				t.Fatalf("registration returned an invalid user: %v", err)
			}
			pair := tokens(request(http.MethodPost, "/auth/login", credentials, "", http.StatusOK))
			profile := request(http.MethodGet, "/users/profile", nil, pair.AccessToken, http.StatusOK)
			var authenticated dto.UserResponse
			if err := json.Unmarshal(profile, &authenticated); err != nil || authenticated.ID != user.ID {
				t.Fatalf("profile does not identify the registered user: %v", err)
			}
			request(http.MethodGet, "/users", nil, pair.AccessToken, http.StatusForbidden)
			if transport == "cookie" {
				for _, origin := range []string{"", "null", "https://untrusted.example.com"} {
					req, err := http.NewRequest(http.MethodPut, server.URL+"/api/v1/users/profile", bytes.NewBufferString(`{"name":"Forged update"}`))
					if err != nil {
						t.Fatal(err)
					}
					req.Header.Set("Content-Type", "application/json")
					if origin != "" {
						req.Header.Set("Origin", origin)
					}
					resp, err := client.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					_ = resp.Body.Close()
					if resp.StatusCode != http.StatusForbidden {
						t.Fatalf("cookie-authenticated write with Origin %q: status=%d, want 403", origin, resp.StatusCode)
					}
				}
			}
			var refreshPayload any
			if transport == "body" {
				refreshPayload = dto.RefreshRequest{RefreshToken: pair.RefreshToken}
			}
			rotated := tokens(request(http.MethodPost, "/auth/refresh", refreshPayload, "", http.StatusOK))
			if rotated.AccessToken == pair.AccessToken || rotated.RefreshToken == pair.RefreshToken {
				t.Fatal("refresh did not replace both tokens")
			}
			request(http.MethodGet, "/users/profile", nil, rotated.AccessToken, http.StatusOK)
			currentCookies := jar.Cookies(authURL)
			var logoutPayload any
			if transport == "body" {
				logoutPayload = dto.LogoutRequest{RefreshToken: rotated.RefreshToken}
			}
			request(http.MethodPost, "/auth/logout", logoutPayload, rotated.AccessToken, http.StatusOK)
			if transport == "cookie" && len(jar.Cookies(authURL)) != 0 {
				t.Fatal("logout did not clear authentication cookies")
			}
			// Replay the logged-out credentials to verify server-side revocation.
			jar.SetCookies(authURL, currentCookies)
			request(http.MethodGet, "/users/profile", nil, rotated.AccessToken, http.StatusUnauthorized)
			if transport == "body" {
				refreshPayload = dto.RefreshRequest{RefreshToken: rotated.RefreshToken}
			}
			request(http.MethodPost, "/auth/refresh", refreshPayload, "", http.StatusUnauthorized)
		})
	}
}
