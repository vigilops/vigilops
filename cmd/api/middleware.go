package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/keelwave/keelwave/internal/auth"
	"github.com/keelwave/keelwave/internal/store"
)

const (
	RoleLevelMember = 1
	RoleLevelAdmin  = 2
	RoleLevelOwner  = 3
)

var orgRoleLevels = map[string]int{
	"member": RoleLevelMember,
	"admin":  RoleLevelAdmin,
	"owner":  RoleLevelOwner,
}

func orgRoleLevel(role string) int {
	if l, ok := orgRoleLevels[role]; ok {
		return l
	}
	return 0
}

func (app *application) requireOrgRoleMiddleware(minLevel int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, err := parseUUIDParam(r, "orgID")
			if err != nil {
				app.badRequestResponse(w, r, err)
				return
			}
			userID := userIDFromContext(r.Context())
			m, err := app.store.OrganizationMembers.Get(r.Context(), orgID, userID)
			if err != nil {
				switch err {
				case store.ErrNotFound:
					app.forbiddenResponse(w, r)
				default:
					app.internalServerError(w, r, err)
				}
				return
			}
			if orgRoleLevel(m.Role) < minLevel {
				app.forbiddenResponse(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (app *application) apiKeyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := extractKey(r)
		plaintext, err := auth.Parse(raw)
		if err != nil {
			app.unauthorizedResponse(w, r, err)
			return
		}

		k, err := app.store.APIKeys.GetByHash(r.Context(), auth.Hash(plaintext))
		if err != nil {
			switch err {
			case store.ErrNotFound:
				app.unauthorizedResponse(w, r, err)
			default:
				app.internalServerError(w, r, err)
			}
			return
		}

		go app.touchKey(k.ID)

		// If the route has a {projectID} param it must match the key's project.
		// Ingest routes have no such param, so URLParam returns "" and the check is skipped.
		if urlProjID := chi.URLParam(r, "projectID"); urlProjID != "" && urlProjID != k.ProjectID.String() {
			app.notFoundResponse(w, r, store.ErrNotFound)
			return
		}

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

// sessionFromCookie validates the session cookie and returns the session.
// On failure it writes an error response and returns nil.
func (app *application) sessionFromCookie(w http.ResponseWriter, r *http.Request) *store.Session {
	cookie, err := r.Cookie(app.config.auth.cookieName)
	if err != nil {
		app.unauthorizedResponse(w, r, errors.New("session required"))
		return nil
	}
	sess, err := app.store.Sessions.GetByHash(r.Context(), auth.HashSession(cookie.Value))
	if err != nil {
		switch err {
		case store.ErrNotFound:
			app.unauthorizedResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return nil
	}
	go app.touchSession(sess.ID)
	return sess
}

func (app *application) userAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := app.sessionFromCookie(w, r)
		if sess == nil {
			return
		}
		ctx := withUserID(r.Context(), sess.UserID)
		ctx = withSessionID(ctx, sess.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) requireVerifiedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := userIDFromContext(r.Context())
		if userID == uuid.Nil {
			// API key path — no user to verify.
			next.ServeHTTP(w, r)
			return
		}
		u, err := app.store.Users.GetByID(r.Context(), userID)
		if err != nil {
			switch err {
			case store.ErrNotFound:
				app.unauthorizedResponse(w, r, err)
			default:
				app.internalServerError(w, r, err)
			}
			return
		}
		if !u.IsVerified {
			_ = writeJSONError(w, http.StatusForbidden, "email not verified — check your inbox")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) touchSession(id uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.store.Sessions.TouchLastUsed(ctx, id); err != nil {
		app.logger.Warnw("touch session last_used_at failed", "err", err, "session_id", id)
	}
}

func (app *application) projectMemberAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := app.sessionFromCookie(w, r)
		if sess == nil {
			return
		}

		projectID, err := parseUUIDParam(r, "projectID")
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
		proj, err := app.store.Projects.GetByID(r.Context(), projectID)
		if err != nil {
			switch err {
			case store.ErrNotFound:
				app.notFoundResponse(w, r, err)
			default:
				app.internalServerError(w, r, err)
			}
			return
		}
		_, err = app.store.OrganizationMembers.Get(r.Context(), proj.OrganizationID, sess.UserID)
		if err != nil {
			switch err {
			case store.ErrNotFound:
				app.forbiddenResponse(w, r)
			default:
				app.internalServerError(w, r, err)
			}
			return
		}
		ctx := withProjectID(r.Context(), projectID)
		ctx = withUserID(ctx, sess.UserID)
		ctx = withSessionID(ctx, sess.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// queryAuthMiddleware accepts either an API key (SDK/programmatic) or a session cookie
// (dashboard). The two paths are exclusive — a request with X-API-Key never
// falls through to the session path on failure.
func (app *application) queryAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if extractKey(r) != "" {
			app.apiKeyAuthMiddleware(next).ServeHTTP(w, r)
			return
		}
		app.projectMemberAuthMiddleware(next).ServeHTTP(w, r)
	})
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
