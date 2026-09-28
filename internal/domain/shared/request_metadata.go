package shared

import (
	"context"

	"uuid"
)

type requestMetadataKey uint8

const (
	actorIDKey requestMetadataKey = iota
	requestIDKey
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

// WithRequestID adds a request correlation ID to a request context.
func WithRequestID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request correlation ID, when the call has one.
func RequestIDFromContext(ctx context.Context) *uuid.UUID {
	id, ok := ctx.Value(requestIDKey).(uuid.UUID)
	if !ok {
		return nil
	}
	return &id
}
