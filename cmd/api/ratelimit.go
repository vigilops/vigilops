package main

import (
	"net/http"

	"github.com/go-chi/httprate"
	"github.com/google/uuid"
)

// ingestKeyFunc identifies the rate-limit bucket for the per-API-key
// layer. Pulls api_key_id from the request context populated by
// apiKeyAuth. Falls open with the empty string if missing (same posture
// as projectIDFromContext) so a wiring bug becomes one shared bucket
// rather than a panic.
func (app *application) ingestKeyFunc(r *http.Request) (string, error) {
	id := apiKeyIDFromContext(r.Context())
	if id == uuid.Nil {
		return "", nil
	}
	return id.String(), nil
}

func (app *application) ingestIPRateLimit() func(http.Handler) http.Handler {
	return httprate.Limit(
		app.config.rateLimit.ingestIPPerMinute,
		app.config.rateLimit.ingestWindow,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(app.rateLimitHandlerAdapter),
	)
}

func (app *application) ingestKeyRateLimit() func(http.Handler) http.Handler {
	return httprate.Limit(
		app.config.rateLimit.ingestKeyPerMinute,
		app.config.rateLimit.ingestWindow,
		httprate.WithKeyFuncs(app.ingestKeyFunc),
		httprate.WithLimitHandler(app.rateLimitHandlerAdapter),
	)
}

// rateLimitHandlerAdapter bridges httprate's limit-exceeded callback to
// the project's standard error envelope. httprate has already set the
// Retry-After + X-RateLimit-* headers on w by the time this runs;
// reading Retry-After back off w.Header() lets rateLimitResponse re-emit
// it alongside the JSON body.
func (app *application) rateLimitHandlerAdapter(w http.ResponseWriter, r *http.Request) {
	app.rateLimitResponse(w, r, w.Header().Get("Retry-After"))
}
