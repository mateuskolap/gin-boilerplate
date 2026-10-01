package config

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		Env: "test", DBHost: "localhost", DBPort: 5432, DBUser: "postgres",
		DBPassword: "secret", DBName: "app", DBSSLMode: "disable",
		AuthTokenTransport: "body", AuthCookieSameSite: "lax", Port: 8080,
		DBMaxOpenConnections: 10, DBMaxIdleConnections: 5,
		DBConnectionLifetime: time.Minute, DBConnectionIdleTime: time.Minute,
		RedisHost: "localhost", RedisPort: 6379, QueueRedisDB: 1, QueueConcurrency: 1,
		PasswordValidationLevel: 1,
		QueueShutdownTimeout:    time.Minute, RefreshTokenRetention: time.Hour,
		JWTSecret: strings.Repeat("s", 32), JWTIssuer: "issuer", JWTAudience: "audience",
		JWTExpiration: time.Minute, RefreshExpiration: time.Hour, StorageRoot: "./storage",
		SMTPPort: 587, ReadHeaderTimeout: time.Second, ReadTimeout: time.Second,
		WriteTimeout: time.Second, IdleTimeout: time.Second, ShutdownTimeout: time.Second,
	}
}

func TestConfigValidateTokenTransport(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Config)
		wantErr string
	}{
		{name: "body transport"},
		{name: "cookie transport with explicit origin", setup: func(c *Config) {
			c.AuthTokenTransport = "cookie"
			c.CORSAllowedOrigins = []string{"https://app.example.com"}
		}},
		{name: "unknown transport", setup: func(c *Config) { c.AuthTokenTransport = "header" }, wantErr: "AUTH_TOKEN_TRANSPORT"},
		{name: "cross-site cookies", setup: func(c *Config) {
			c.AuthTokenTransport = "cookie"
			c.AuthCookieSameSite = "none"
			c.CORSAllowedOrigins = []string{"https://frontend.example.com"}
		}},
		{name: "strict cookies", setup: func(c *Config) { c.AuthCookieSameSite = "strict" }},
		{name: "invalid same-site", setup: func(c *Config) { c.AuthCookieSameSite = "invalid" }, wantErr: "AUTH_COOKIE_SAME_SITE"},
		{name: "cookies without origins", setup: func(c *Config) { c.AuthTokenTransport = "cookie" }, wantErr: "explicit origins"},
		{name: "wildcard origin with cookies", setup: func(c *Config) {
			c.AuthTokenTransport = "cookie"
			c.CORSAllowedOrigins = []string{"*"}
		}, wantErr: "explicit origins"},
	}
	for _, origin := range []string{"null", "", "https://*.example.com", "https://app.example.com/", "https://app.example.com?", "https://app.example.com#", "https://app.example.com?query=1", "https://app.example.com#fragment", "https://user@app.example.com", "ftp://app.example.com"} {
		tests = append(tests, struct {
			name    string
			setup   func(*Config)
			wantErr string
		}{name: "invalid cookie origin " + origin, setup: func(c *Config) {
			c.AuthTokenTransport = "cookie"
			c.CORSAllowedOrigins = []string{origin}
		}, wantErr: "explicit origins"})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			if tt.setup != nil {
				tt.setup(&cfg)
			}
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSeederRequiresStrongPassword(t *testing.T) {
	for _, password := range []string{"", "short", "long-enough-password"} {
		t.Run(password, func(t *testing.T) {
			cfg := validConfig()
			cfg.AdminPassword = password
			err := cfg.ValidateSeeder()
			if (len(password) >= 12) != (err == nil) {
				t.Fatalf("ValidateSeeder() error = %v for password length %d", err, len(password))
			}
		})
	}
}

func TestConfigValidateRejectsUnsafeAndOutOfRangeValues(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*Config)
		wantErr string
	}{
		{name: "empty redis host", change: func(c *Config) { c.RedisHost = " " }, wantErr: "REDIS_HOST"},
		{name: "short jwt secret", change: func(c *Config) { c.JWTSecret = "short" }, wantErr: "JWT_SECRET"},
		{name: "empty issuer", change: func(c *Config) { c.JWTIssuer = " " }, wantErr: "JWT_ISSUER"},
		{name: "empty audience", change: func(c *Config) { c.JWTAudience = " " }, wantErr: "JWT_AUDIENCE"},
		{name: "invalid jwt expiry", change: func(c *Config) { c.JWTExpiration = 0 }, wantErr: "JWT_EXPIRATION"},
		{name: "invalid password validation level low", change: func(c *Config) { c.PasswordValidationLevel = 0 }, wantErr: "PASSWORD_VALIDATION_LEVEL"},
		{name: "invalid password validation level high", change: func(c *Config) { c.PasswordValidationLevel = 4 }, wantErr: "PASSWORD_VALIDATION_LEVEL"},
		{name: "invalid refresh expiry", change: func(c *Config) { c.RefreshExpiration = 0 }, wantErr: "REFRESH_EXPIRATION"},
		{name: "empty storage root", change: func(c *Config) { c.StorageRoot = " " }, wantErr: "STORAGE_ROOT"},
		{name: "invalid api port", change: func(c *Config) { c.Port = 65536 }, wantErr: "PORT"},
		{name: "invalid redis port", change: func(c *Config) { c.RedisPort = 0 }, wantErr: "REDIS_PORT"},
		{name: "negative redis database", change: func(c *Config) { c.RedisDB = -1 }, wantErr: "must not be negative"},
		{name: "same redis databases", change: func(c *Config) { c.QueueRedisDB = c.RedisDB }, wantErr: "must differ"},
		{name: "invalid queue concurrency", change: func(c *Config) { c.QueueConcurrency = 0 }, wantErr: "QUEUE_CONCURRENCY"},
		{name: "invalid queue timeout", change: func(c *Config) { c.QueueShutdownTimeout = 0 }, wantErr: "queue and refresh token"},
		{name: "invalid retention", change: func(c *Config) { c.RefreshTokenRetention = 0 }, wantErr: "queue and refresh token"},
		{name: "invalid smtp port", change: func(c *Config) { c.SMTPPort = 0 }, wantErr: "SMTP_PORT"},
		{name: "no open database connections", change: func(c *Config) { c.DBMaxOpenConnections = 0 }, wantErr: "DB_MAX_OPEN_CONNECTIONS"},
		{name: "too many idle connections", change: func(c *Config) { c.DBMaxIdleConnections = 11 }, wantErr: "DB_MAX_IDLE_CONNECTIONS"},
		{name: "invalid connection lifetime", change: func(c *Config) { c.DBConnectionLifetime = 0 }, wantErr: "database connection durations"},
		{name: "invalid http timeout", change: func(c *Config) { c.WriteTimeout = 0 }, wantErr: "HTTP timeouts"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			tc.change(&cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestDatabaseConfigValidationURLAndProductionDefaults(t *testing.T) {
	cfg := DatabaseConfig{Env: "production", DBHost: "::1", DBPort: 5432, DBUser: "api user", DBPassword: "p@ss/word", DBName: "app", DBSSLMode: "verify-full"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	parsed, err := url.Parse(cfg.URL())
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	if parsed.Scheme != "postgres" || parsed.Host != "[::1]:5432" || parsed.User.Username() != "api user" || password != "p@ss/word" || parsed.Path != "/app" || parsed.Query().Get("sslmode") != "verify-full" {
		t.Fatalf("URL() = %q", parsed.String())
	}

	invalid := []struct {
		name   string
		change func(*DatabaseConfig)
	}{
		{name: "environment", change: func(c *DatabaseConfig) { c.Env = "staging" }},
		{name: "missing host", change: func(c *DatabaseConfig) { c.DBHost = " " }},
		{name: "port", change: func(c *DatabaseConfig) { c.DBPort = 70000 }},
		{name: "ssl mode", change: func(c *DatabaseConfig) { c.DBSSLMode = "require" }},
		{name: "development password in production", change: func(c *DatabaseConfig) { c.DBPassword = "postgres" }},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			candidate := cfg
			tc.change(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid database config")
			}
		})
	}
}

func TestLoadConfigNormalizesAdminEmailAndProductionSSLMode(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_PASSWORD", "production-password")
	t.Setenv("DB_SSLMODE", "")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("AUTH_TOKEN_TRANSPORT", "body")
	t.Setenv("PASSWORD_VALIDATION_LEVEL", "2")
	t.Setenv("ADMIN_EMAIL", "  ADMIN@Example.COM  ")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Env != "production" || cfg.DBSSLMode != "verify-full" || cfg.AdminEmail != "admin@example.com" || cfg.PasswordValidationLevel != 2 {
		t.Fatalf("LoadConfig() environment=%q sslmode=%q admin email=%q password level=%d", cfg.Env, cfg.DBSSLMode, cfg.AdminEmail, cfg.PasswordValidationLevel)
	}
}

func TestLoadDatabaseConfigUsesProductionSSLDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_PASSWORD", "production-password")
	t.Setenv("DB_SSLMODE", "")
	cfg, err := LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}
	if cfg.DBSSLMode != "verify-full" {
		t.Fatalf("LoadDatabaseConfig() SSL mode = %q, want verify-full", cfg.DBSSLMode)
	}
}

func TestLoadSeederConfigRequiresOnlyDatabaseAndAdminSettings(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ENVIRONMENT", "test")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("REDIS_PORT", "not-a-number")
	t.Setenv("ADMIN_PASSWORD", "integration-password")
	t.Setenv("ADMIN_EMAIL", " ADMIN@Example.TEST ")
	cfg, err := LoadSeederConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminEmail != "admin@example.test" || cfg.DBMaxOpenConnections != 1 {
		t.Fatalf("unexpected seed settings: email=%q connections=%d", cfg.AdminEmail, cfg.DBMaxOpenConnections)
	}
	t.Setenv("ADMIN_PASSWORD", "short")
	if _, err := LoadSeederConfig(); err == nil {
		t.Fatal("seeder accepted a weak administrator password")
	}
}
