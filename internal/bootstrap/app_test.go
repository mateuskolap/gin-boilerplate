package bootstrap

import (
	"crypto/tls"
	"testing"

	"gin-boilerplate/config"
)

func TestNewRedisOptionsEnablesVerifiedTLSOnlyInProduction(t *testing.T) {
	cfg := &config.Config{
		Env:           "production",
		RedisHost:     "redis.example.com",
		RedisPort:     6379,
		RedisPassword: "secret",
	}

	options := newRedisOptions(cfg, 2)
	if options.TLSConfig == nil {
		t.Fatal("production Redis options must enable TLS")
	}
	if options.TLSConfig.MinVersion != tls.VersionTLS12 || options.TLSConfig.ServerName != cfg.RedisHost {
		t.Fatalf("production Redis TLS config = %+v", options.TLSConfig)
	}
	if options.Addr != "redis.example.com:6379" || options.Password != cfg.RedisPassword || options.DB != 2 {
		t.Fatalf("Redis options = %+v", options)
	}

	cfg.Env = "development"
	if options := newRedisOptions(cfg, 0); options.TLSConfig != nil {
		t.Fatalf("development Redis TLS config = %+v, want nil", options.TLSConfig)
	}
}
