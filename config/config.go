package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env               string
	Port              string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	RedisHost         string
	RedisPort         string
	RedisPassword     string
	RedisDB           int
	JWTSecret         string
	JWTExpiration     time.Duration
	RefreshExpiration time.Duration
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, loading configuration from environment variables")
	}

	env := getEnv("ENVIRONMENT", "development")

	defaultSSLMode := "disable"
	if env == "production" {
		defaultSSLMode = "require"
	}

	jwtExpMinutes, err := strconv.Atoi(getEnv("JWT_EXPIRATION_MINUTES", "10"))
	if err != nil || jwtExpMinutes <= 0 {
		jwtExpMinutes = 10
	}

	refreshExpMinutes, err := strconv.Atoi(getEnv("REFRESH_EXPIRATION_MINUTES", "1440"))
	if err != nil || refreshExpMinutes <= 0 {
		refreshExpMinutes = 1440
	}

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	jwtSecret := getEnv("JWT_SECRET", "gin-boilerplate-super-secure-jwt-secret-key-32-bytes!")
	if len(jwtSecret) < 32 {
		log.Fatalf("FATAL: JWT_SECRET must be at least 32 characters long, got %d characters", len(jwtSecret))
	}

	return &Config{
		Env:               env,
		Port:              getEnv("PORT", "8080"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "postgres"),
		DBName:            getEnv("DB_NAME", "boilerplate"),
		DBSSLMode:         getEnv("DB_SSLMODE", defaultSSLMode),
		RedisHost:         getEnv("REDIS_HOST", "localhost"),
		RedisPort:         getEnv("REDIS_PORT", "6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           redisDB,
		JWTSecret:         jwtSecret,
		JWTExpiration:     time.Minute * time.Duration(jwtExpMinutes),
		RefreshExpiration: time.Minute * time.Duration(refreshExpMinutes),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
