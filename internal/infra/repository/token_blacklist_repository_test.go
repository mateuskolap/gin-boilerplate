package repository

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memoryCache struct {
	values map[string]any
	setKey string
	setTTL time.Duration
	err    error
}

func (c *memoryCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	if c.err != nil {
		return c.err
	}
	if c.values == nil {
		c.values = make(map[string]any)
	}
	c.setKey, c.setTTL = key, ttl
	c.values[key] = value
	return nil
}

func (c *memoryCache) Get(_ context.Context, key string, dest any) error {
	if c.err != nil {
		return c.err
	}
	value := c.values[key]
	switch target := dest.(type) {
	case *string:
		if stored, ok := value.(string); ok {
			*target = stored
		}
	case *int64:
		if stored, ok := value.(int64); ok {
			*target = stored
		}
	default:
		return errors.New("unexpected cache target")
	}
	return nil
}

func TestTokenBlacklistStoresAndChecksAccessAndUserRevocations(t *testing.T) {
	ctx := context.Background()
	cache := &memoryCache{}
	blacklist := NewTokenBlackListRepository(cache)
	ttl := 15 * time.Minute
	if err := blacklist.RevokeToken(ctx, "jti-123", ttl); err != nil || cache.setKey != "blacklist:jti:jti-123" || cache.setTTL != ttl {
		t.Fatalf("RevokeToken() key=%q ttl=%s error=%v", cache.setKey, cache.setTTL, err)
	}
	if revoked, err := blacklist.IsRevoked(ctx, "jti-123"); err != nil || !revoked {
		t.Fatalf("IsRevoked() revoked=%v error=%v", revoked, err)
	}
	if revoked, err := blacklist.IsRevoked(ctx, "missing"); err != nil || revoked {
		t.Fatalf("IsRevoked(missing) revoked=%v error=%v", revoked, err)
	}

	if err := blacklist.RevokeUserTokens(ctx, "user-123", time.Hour); err != nil || cache.setKey != "blacklist:user:user-123" {
		t.Fatalf("RevokeUserTokens() key=%q error=%v", cache.setKey, err)
	}
	revokedAt := cache.values["blacklist:user:user-123"].(int64)
	if revoked, err := blacklist.IsUserTokenRevoked(ctx, "user-123", time.Unix(revokedAt-1, 0)); err != nil || !revoked {
		t.Fatalf("IsUserTokenRevoked(old) revoked=%v error=%v", revoked, err)
	}
	if revoked, err := blacklist.IsUserTokenRevoked(ctx, "user-123", time.Unix(revokedAt+1, 0)); err != nil || revoked {
		t.Fatalf("IsUserTokenRevoked(new) revoked=%v error=%v", revoked, err)
	}
	if revoked, err := blacklist.IsUserTokenRevoked(ctx, "unknown-user", time.Now()); err != nil || revoked {
		t.Fatalf("IsUserTokenRevoked(unknown) revoked=%v error=%v", revoked, err)
	}
}

func TestTokenBlacklistPropagatesCacheFailures(t *testing.T) {
	want := errors.New("cache unavailable")
	blacklist := NewTokenBlackListRepository(&memoryCache{err: want})
	if err := blacklist.RevokeToken(context.Background(), "jti", time.Minute); !errors.Is(err, want) {
		t.Fatalf("RevokeToken() error = %v", err)
	}
	if _, err := blacklist.IsRevoked(context.Background(), "jti"); !errors.Is(err, want) {
		t.Fatalf("IsRevoked() error = %v", err)
	}
	if _, err := blacklist.IsUserTokenRevoked(context.Background(), "user", time.Now()); !errors.Is(err, want) {
		t.Fatalf("IsUserTokenRevoked() error = %v", err)
	}
}
