package domain

import (
	"context"
	"time"
)

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

const (
	// ContextKeyRequestID is the key for request ID in context
	ContextKeyRequestID ContextKey = "request_id"
	// ContextKeyUserID is the key for user ID in context
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyLogger is the key for logger in context
	ContextKeyLogger ContextKey = "logger"
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// RequestID retrieves the request ID from context
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// UserID retrieves the user ID from context
func UserID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyUserID).(string); ok {
		return id
	}
	return ""
}

// WithTimeout creates a context with timeout
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// WithDeadline creates a context with deadline
func WithDeadline(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, deadline)
}

