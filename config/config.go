package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseConfig
	Port                  int           `env:"PORT" envDefault:"8080"`
	DBMaxOpenConnections  int           `env:"DB_MAX_OPEN_CONNECTIONS" envDefault:"25"`
	DBMaxIdleConnections  int           `env:"DB_MAX_IDLE_CONNECTIONS" envDefault:"5"`
	DBConnectionLifetime  time.Duration `env:"DB_CONNECTION_LIFETIME" envDefault:"30m"`
	DBConnectionIdleTime  time.Duration `env:"DB_CONNECTION_IDLE_TIME" envDefault:"5m"`
	RedisHost             string        `env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort             int           `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword         string        `env:"REDIS_PASSWORD"`
	RedisDB               int           `env:"REDIS_DB" envDefault:"0"`
	QueueRedisDB          int           `env:"QUEUE_REDIS_DB" envDefault:"1"`
	QueueConcurrency      int           `env:"QUEUE_CONCURRENCY" envDefault:"10"`
	QueueShutdownTimeout  time.Duration `env:"QUEUE_SHUTDOWN_TIMEOUT" envDefault:"30s"`
	RefreshTokenRetention time.Duration `env:"REFRESH_TOKEN_RETENTION" envDefault:"720h"`
	JWTSecret             string        `env:"JWT_SECRET,required,notEmpty"`
	JWTIssuer             string        `env:"JWT_ISSUER" envDefault:"gin-boilerplate"`
	JWTAudience           string        `env:"JWT_AUDIENCE" envDefault:"gin-boilerplate-api"`
	JWTExpiration         time.Duration `env:"JWT_EXPIRATION" envDefault:"10m"`
	RefreshExpiration     time.Duration `env:"REFRESH_EXPIRATION" envDefault:"24h"`
	StorageRoot           string        `env:"STORAGE_ROOT" envDefault:"./storage/private"`
	SMTPHost              string        `env:"SMTP_HOST"`
	SMTPPort              int           `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser              string        `env:"SMTP_USER"`
	SMTPPassword          string        `env:"SMTP_PASSWORD"`
	SMTPFrom              string        `env:"SMTP_FROM"`
	AdminName             string        `env:"ADMIN_NAME" envDefault:"Admin"`
	AdminEmail            string        `env:"ADMIN_EMAIL" envDefault:"admin@example.com"`
	AdminPassword         string        `env:"ADMIN_PASSWORD"`
	TrustedProxies        []string      `env:"TRUSTED_PROXIES" envSeparator:","`
	CORSAllowedOrigins    []string      `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`
	ReadHeaderTimeout     time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout           time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout          time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout           time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout       time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

type DatabaseConfig struct {
	Env        string `env:"ENVIRONMENT" envDefault:"development"`
	DBHost     string `env:"DB_HOST" envDefault:"localhost"`
	DBPort     int    `env:"DB_PORT" envDefault:"5432"`
	DBUser     string `env:"DB_USER" envDefault:"postgres"`
	DBPassword string `env:"DB_PASSWORD" envDefault:"postgres"`
	DBName     string `env:"DB_NAME" envDefault:"boilerplate"`
	DBSSLMode  string `env:"DB_SSLMODE"`
}

func LoadDatabaseConfig() (*DatabaseConfig, error) {
	_ = godotenv.Load()

	cfg, err := env.ParseAs[DatabaseConfig]()
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}

	cfg.setDefaultSSLMode()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("parse configuration: %w", err)
	}

	cfg.DatabaseConfig.setDefaultSSLMode()

	cfg.AdminEmail = strings.ToLower(strings.TrimSpace(cfg.AdminEmail))
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	var errs []error

	errs = append(errs, c.DatabaseConfig.Validate())
	if strings.TrimSpace(c.RedisHost) == "" {
		errs = append(errs, fmt.Errorf("REDIS_HOST must not be empty"))
	}
	if len(c.JWTSecret) < 32 {
		errs = append(errs, fmt.Errorf("JWT_SECRET must contain at least 32 characters"))
	}
	if strings.TrimSpace(c.JWTIssuer) == "" {
		errs = append(errs, fmt.Errorf("JWT_ISSUER must not be empty"))
	}
	if strings.TrimSpace(c.JWTAudience) == "" {
		errs = append(errs, fmt.Errorf("JWT_AUDIENCE must not be empty"))
	}
	if c.JWTExpiration <= 0 {
		errs = append(errs, fmt.Errorf("JWT_EXPIRATION must be greater than zero"))
	}
	if c.RefreshExpiration <= 0 {
		errs = append(errs, fmt.Errorf("REFRESH_EXPIRATION must be greater than zero"))
	}
	if strings.TrimSpace(c.StorageRoot) == "" {
		errs = append(errs, fmt.Errorf("STORAGE_ROOT must not be empty"))
	}
	if c.Port <= 0 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("PORT must be between 1 and 65535"))
	}
	if c.RedisPort <= 0 || c.RedisPort > 65535 {
		errs = append(errs, fmt.Errorf("REDIS_PORT must be between 1 and 65535"))
	}
	if c.RedisDB < 0 || c.QueueRedisDB < 0 {
		errs = append(errs, fmt.Errorf("Redis database numbers must not be negative"))
	}
	if c.QueueRedisDB == c.RedisDB {
		errs = append(errs, fmt.Errorf("QUEUE_REDIS_DB must differ from REDIS_DB"))
	}
	if c.QueueConcurrency <= 0 {
		errs = append(errs, fmt.Errorf("QUEUE_CONCURRENCY must be greater than zero"))
	}
	if c.QueueShutdownTimeout <= 0 || c.RefreshTokenRetention <= 0 {
		errs = append(errs, fmt.Errorf("queue and refresh token retention durations must be greater than zero"))
	}
	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		errs = append(errs, fmt.Errorf("SMTP_PORT must be between 1 and 65535"))
	}
	if c.DBMaxOpenConnections <= 0 {
		errs = append(errs, fmt.Errorf("DB_MAX_OPEN_CONNECTIONS must be greater than zero"))
	}
	if c.DBMaxIdleConnections < 0 || c.DBMaxIdleConnections > c.DBMaxOpenConnections {
		errs = append(errs, fmt.Errorf("DB_MAX_IDLE_CONNECTIONS must be between 0 and DB_MAX_OPEN_CONNECTIONS"))
	}
	if c.DBConnectionLifetime <= 0 || c.DBConnectionIdleTime <= 0 {
		errs = append(errs, fmt.Errorf("database connection durations must be greater than zero"))
	}
	if c.ReadHeaderTimeout <= 0 || c.ReadTimeout <= 0 || c.WriteTimeout <= 0 || c.IdleTimeout <= 0 || c.ShutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("HTTP timeouts must be greater than zero"))
	}
	return errors.Join(errs...)
}

func (c *DatabaseConfig) setDefaultSSLMode() {
	if c.DBSSLMode == "" {
		c.DBSSLMode = "disable"
		if c.Env == "production" {
			c.DBSSLMode = "require"
		}
	}
}

func (c *DatabaseConfig) Validate() error {
	var errs []error
	if c.Env != "development" && c.Env != "production" && c.Env != "test" {
		errs = append(errs, fmt.Errorf("ENVIRONMENT must be development, production, or test"))
	}
	if strings.TrimSpace(c.DBHost) == "" || strings.TrimSpace(c.DBUser) == "" || strings.TrimSpace(c.DBName) == "" {
		errs = append(errs, fmt.Errorf("database host, user, and name must not be empty"))
	}
	if c.DBPort <= 0 || c.DBPort > 65535 {
		errs = append(errs, fmt.Errorf("DB_PORT must be between 1 and 65535"))
	}
	switch c.DBSSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		errs = append(errs, fmt.Errorf("DB_SSLMODE is invalid"))
	}
	if c.Env == "production" && c.DBPassword == "postgres" {
		errs = append(errs, fmt.Errorf("DB_PASSWORD must not use the development default in production"))
	}
	return errors.Join(errs...)
}

func (c *DatabaseConfig) URL() string {
	connectionURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   net.JoinHostPort(c.DBHost, strconv.Itoa(c.DBPort)),
		Path:   c.DBName,
	}
	query := connectionURL.Query()
	query.Set("sslmode", c.DBSSLMode)
	connectionURL.RawQuery = query.Encode()
	return connectionURL.String()
}

func (c *Config) ValidateSeeder() error {
	if strings.TrimSpace(c.AdminPassword) == "" {
		return fmt.Errorf("ADMIN_PASSWORD is required when running seeders")
	}
	if len(c.AdminPassword) < 12 {
		return fmt.Errorf("ADMIN_PASSWORD must contain at least 12 characters")
	}
	return nil
}
