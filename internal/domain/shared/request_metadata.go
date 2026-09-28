package shared

import (
	"context"

	"uuid"
)

type requestMetadataKey uint8

const (
	actorIDKey requestMetadataKey = iota
	requestIPKey
)

// WithActorID adds the authenticated user ID to a request context.
func WithActorID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, actorIDKey, id)
}

// ActorIDFromContext returns the authenticated user ID, when the call has one.
func ActorIDFromContext(ctx context.Context) *uuid.UUID {
	id, ok := ctx.Value(actorIDKey).(uuid.UUID)
	if !ok {
		return nil
	}
	return &id
}

// WithRequestIP adds the client IP address to a request context.
func WithRequestIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, requestIPKey, ip)
}

// RequestIPFromContext returns the client IP address, when the call has one.
func RequestIPFromContext(ctx context.Context) *string {
	ip, ok := ctx.Value(requestIPKey).(string)
	if !ok || ip == "" {
		return nil
	}
	return &ip
}
