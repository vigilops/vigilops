package main

import (
	"net/http"

	"github.com/go-chi/httprate"
	"github.com/google/uuid"
)

// Returns empty string if api_key_id is missing from ctx — that means
// the route was mounted without apiKeyAuthMiddleware (wiring bug). httprate keys
// requests by this string, so all anomalous requests share one
// rate-limit counter rather than panicking the handler.
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

// httprate has already set Retry-After + X-RateLimit-* on w; read it
// back so rateLimitResponse re-emits it with the standard JSON envelope.
func (app *application) rateLimitHandlerAdapter(w http.ResponseWriter, r *http.Request) {
	app.rateLimitResponse(w, r, w.Header().Get("Retry-After"))
}
