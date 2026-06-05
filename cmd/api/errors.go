package main

import "net/http"

func (app *application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("internal server error",
		"method", r.Method, "path", r.URL.Path, "err", err)
	_ = writeJSONError(w, http.StatusInternalServerError, "internal server error")
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("bad request",
		"method", r.Method, "path", r.URL.Path, "err", err)
	_ = writeJSONError(w, http.StatusBadRequest, err.Error())
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("not found",
		"method", r.Method, "path", r.URL.Path, "err", err)
	_ = writeJSONError(w, http.StatusNotFound, "not found")
}

func (app *application) conflictResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("conflict",
		"method", r.Method, "path", r.URL.Path, "err", err)
	_ = writeJSONError(w, http.StatusConflict, err.Error())
}

func (app *application) unauthorizedResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("unauthorized",
		"method", r.Method, "path", r.URL.Path, "err", err)
	_ = writeJSONError(w, http.StatusUnauthorized, "unauthorized")
}

func (app *application) forbiddenResponse(w http.ResponseWriter, r *http.Request) {
	app.logger.Warnw("forbidden",
		"method", r.Method, "path", r.URL.Path)
	_ = writeJSONError(w, http.StatusForbidden, "forbidden")
}

func (app *application) rateLimitResponse(w http.ResponseWriter, r *http.Request, retryAfter string) {
	app.logger.Warnw("rate limited",
		"method", r.Method, "path", r.URL.Path)
	if retryAfter != "" {
		w.Header().Set("Retry-After", retryAfter)
	}
	_ = writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
}
