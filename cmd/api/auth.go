package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/keelwave/keelwave/internal/auth"
	"github.com/keelwave/keelwave/internal/mailer"
	"github.com/keelwave/keelwave/internal/store"
)

type registerPayload struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=200"`
	Name     string `json:"name" validate:"max=100"`
}

type loginPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type authUserResponse struct {
	User          *store.User           `json:"user"`
	Organizations []*store.Organization `json:"organizations"`
}

// Register godoc
//
//	@Summary	Register a new user
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		registerPayload	true	"Registration payload"
//	@Success	201		{object}	authUserResponse
//	@Failure	400		{object}	error
//	@Failure	409		{object}	error
//	@Failure	500		{object}	error
//	@Router		/auth/register [post]
func (app *application) registerHandler(w http.ResponseWriter, r *http.Request) {
	var payload registerPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	u := &store.User{Email: payload.Email, Name: payload.Name}
	if err := u.Password.Set(payload.Password); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	plaintext, hash, err := auth.GenerateSession()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	v := &store.UserVerification{
		Token:     hash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := app.store.Users.CreateWithVerification(r.Context(), u, v); err != nil {
		switch err {
		case store.ErrConflict:
			app.conflictResponse(w, r, errors.New("email already registered"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	go func() {
		verifyURL := app.config.auth.dashboardURL + "/verify-email/" + plaintext
		data := struct{ VerifyURL string }{VerifyURL: verifyURL}
		if err := app.mailer.Send(mailer.VerifyTemplate, u.Email, data); err != nil {
			app.logger.Warnw("verification email failed", "to", u.Email, "err", err)
		}
	}()

	if err := app.issueSession(r.Context(), w, u.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, &authUserResponse{
		User: u, Organizations: []*store.Organization{},
	}); err != nil {
		app.internalServerError(w, r, err)
	}
}

// VerifyEmail godoc
//
//	@Summary	Verify email via token from the verification link
//	@Tags		auth
//	@Param		token	path	string	true	"Verification token"
//	@Success	204
//	@Failure	400	{object}	error
//	@Failure	404	{object}	error
//	@Router		/auth/verify-email/{token} [get]
func (app *application) verifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		app.badRequestResponse(w, r, errors.New("token required"))
		return
	}

	hash := auth.HashSession(token)
	if err := app.store.Users.Verify(r.Context(), hash); err != nil {
		switch err {
		case store.ErrNotFound:
			app.notFoundResponse(w, r, errors.New("invalid or expired verification link"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	app.noContentResponse(w)
}

// CreateOrg godoc
//
//	@Summary	Create an organization
//	@Tags		admin/orgs
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		object	true	"name (string, required)"
//	@Success	201		{object}	store.Organization
//	@Failure	400		{object}	error
//	@Failure	500		{object}	error
//	@Router		/admin/orgs [post]
func (app *application) createOrgHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name string `json:"name" validate:"required,min=1,max=100"`
	}
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	userID := userIDFromContext(r.Context())
	org := &store.Organization{Name: payload.Name}
	if err := app.store.Organizations.CreateWithOwner(r.Context(), org, userID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, org); err != nil {
		app.internalServerError(w, r, err)
	}
}

// Login godoc
//
//	@Summary	Log in with email + password
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		loginPayload	true	"Credentials"
//	@Success	200		{object}	authUserResponse
//	@Failure	400		{object}	error
//	@Failure	401		{object}	error
//	@Failure	500		{object}	error
//	@Router		/auth/login [post]
func (app *application) loginHandler(w http.ResponseWriter, r *http.Request) {
	var payload loginPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	u, err := app.store.Users.GetByEmail(r.Context(), payload.Email)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			app.unauthorizedResponse(w, r, errors.New("invalid credentials"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	if !u.Password.Compare(payload.Password) {
		app.unauthorizedResponse(w, r, errors.New("invalid credentials"))
		return
	}

	if err = app.issueSession(r.Context(), w, u.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	organizations, err := app.store.Organizations.ListByUser(r.Context(), u.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, &authUserResponse{
		User: u, Organizations: organizations,
	}); err != nil {
		app.internalServerError(w, r, err)
	}
}

// Logout godoc
//
//	@Summary	End the current session
//	@Tags		auth
//	@Success	204
//	@Failure	500	{object}	error
//	@Router		/auth/logout [post]
func (app *application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if id := sessionIDFromContext(r.Context()); id != uuid.Nil {
		if err := app.store.Sessions.Delete(r.Context(), id); err != nil {
			switch err {
			case store.ErrNotFound:
			default:
				app.internalServerError(w, r, err)
				return
			}
		}
	}
	app.clearSessionCookie(w)
	app.clearOAuthStateCookie(w)
	app.noContentResponse(w)
}

// Me godoc
//
//	@Summary	Get the current user and their organizations
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	authUserResponse
//	@Failure	401	{object}	error
//	@Failure	500	{object}	error
//	@Router		/auth/me [get]
func (app *application) meHandler(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	u, err := app.store.Users.GetByID(r.Context(), userID)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			app.clearSessionCookie(w)
			app.unauthorizedResponse(w, r, errors.New("session invalid"))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	organizations, err := app.store.Organizations.ListByUser(r.Context(), userID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, &authUserResponse{
		User: u, Organizations: organizations,
	}); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) issueSession(ctx context.Context, w http.ResponseWriter, userID uuid.UUID) error {
	plaintext, hash, err := auth.GenerateSession()
	if err != nil {
		return err
	}
	expires := time.Now().Add(app.config.auth.sessionTTL)
	sess := &store.Session{UserID: userID, TokenHash: hash, ExpiresAt: expires}
	if err := app.store.Sessions.Create(ctx, sess); err != nil {
		return err
	}
	app.setSessionCookie(w, plaintext, expires)
	return nil
}

func (app *application) setSessionCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     app.config.auth.cookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   app.config.auth.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (app *application) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     app.config.auth.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   app.config.auth.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (app *application) clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   app.config.auth.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
