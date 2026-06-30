package main

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const (
	projectIDKey ctxKey = iota
	apiKeyIDKey
	userIDKey
	sessionIDKey
)

func withProjectID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, projectIDKey, id)
}

func withAPIKeyID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, apiKeyIDKey, id)
}

func projectIDFromContext(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(projectIDKey).(uuid.UUID)
	return v
}

func apiKeyIDFromContext(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(apiKeyIDKey).(uuid.UUID)
	return v
}

func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func withSessionID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, sessionIDKey, id)
}

func userIDFromContext(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(userIDKey).(uuid.UUID)
	return v
}

func sessionIDFromContext(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(sessionIDKey).(uuid.UUID)
	return v
}
