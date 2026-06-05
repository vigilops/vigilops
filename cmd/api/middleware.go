package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yusufnuru/vigil/internal/auth"
	"github.com/yusufnuru/vigil/internal/store"
)

func (app *application) apiKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := extractKey(r)
		plaintext, err := auth.Parse(raw)
		if err != nil {
			app.unauthorizedResponse(w, r, err)
			return
		}

		k, err := app.store.APIKeys.GetByHash(r.Context(), auth.Hash(plaintext))
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				app.unauthorizedResponse(w, r, err)
				return
			}
			app.internalServerError(w, r, err)
			return
		}

		go app.touchKey(k.ID)

		ctx := withProjectID(r.Context(), k.ProjectID)
		ctx = withAPIKeyID(ctx, k.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) touchKey(id uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.store.APIKeys.TouchLastUsed(ctx, id); err != nil {
		app.logger.Warnw("touch last_used_at failed", "err", err, "key_id", id)
	}
}

func extractKey(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if after, ok := strings.CutPrefix(h, "Bearer "); ok {
			return after
		}
		return h
	}
	return r.Header.Get("X-API-Key")
}
