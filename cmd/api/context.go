package main

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const (
	projectIDKey ctxKey = iota
	apiKeyIDKey
)

func withProjectID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, projectIDKey, id)
}

func withAPIKeyID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, apiKeyIDKey, id)
}

func projectIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(projectIDKey).(uuid.UUID)
	return v, ok
}

func apiKeyIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(apiKeyIDKey).(uuid.UUID)
	return v, ok
}
