package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/keelwave/keelwave/internal/auth"
	"github.com/keelwave/keelwave/internal/store"
)

const (
	oauthStateCookie    = "keelwave_oauth_state"
	oauthRedirectCookie = "keelwave_oauth_redirect"
)

type providerInfo struct {
	id            string
	email         string
	emailVerified bool
	name          string
}

func (app *application) oauthConfig(provider string) *oauth2.Config {
	cb := func(p string) string {
		return fmt.Sprintf("%s/v1/auth/oauth/%s/callback", app.config.auth.publicURL, p)
	}
	switch provider {
	case "google":
		if !app.config.auth.google.configured() {
			return nil
		}
		return &oauth2.Config{
			ClientID:     app.config.auth.google.clientID,
			ClientSecret: app.config.auth.google.clientSecret,
			RedirectURL:  cb("google"),
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		}
	case "github":
		if !app.config.auth.github.configured() {
			return nil
		}
		return &oauth2.Config{
			ClientID:     app.config.auth.github.clientID,
			ClientSecret: app.config.auth.github.clientSecret,
			RedirectURL:  cb("github"),
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		}
	default:
		return nil
	}
}

// OAuthStart godoc
//
//	@Summary	Initiate OAuth login flow
//	@Tags		auth
//	@Param		provider	path	string	true	"OAuth provider (google, github)"
//	@Param		redirect	query	string	false	"Post-login redirect path"
//	@Success	307
//	@Failure	404	{object}	error
//	@Router		/auth/oauth/{provider}/start [get]
func (app *application) oauthStartHandler(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	cfg := app.oauthConfig(provider)
	if cfg == nil {
		app.notFoundResponse(w, r, errors.New("oauth provider not configured"))
		return
	}

	state, _, err := auth.GenerateSession()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   app.config.auth.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	if rp := r.URL.Query().Get("redirect"); strings.HasPrefix(rp, "/") {
		http.SetCookie(w, &http.Cookie{
			Name:     oauthRedirectCookie,
			Value:    rp,
			Path:     "/",
			MaxAge:   600,
			HttpOnly: true,
			Secure:   app.config.auth.cookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
	}

	http.Redirect(w, r, cfg.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

// OAuthCallback godoc
//
//	@Summary	OAuth provider callback — issues session and redirects to dashboard
//	@Tags		auth
//	@Param		provider	path	string	true	"OAuth provider (google, github)"
//	@Param		code		query	string	true	"Authorization code"
//	@Param		state		query	string	true	"CSRF state"
//	@Success	307
//	@Failure	400	{object}	error
//	@Failure	500	{object}	error
//	@Router		/auth/oauth/{provider}/callback [get]
func (app *application) oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	cfg := app.oauthConfig(provider)
	if cfg == nil {
		app.notFoundResponse(w, r, errors.New("oauth provider not configured"))
		return
	}

	// CSRF: the state in the query must match the one we set in the cookie.
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		app.badRequestResponse(w, r, errors.New("invalid oauth state"))
		return
	}

	app.clearOAuthStateCookie(w)

	token, err := cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("oauth exchange: %w", err))
		return
	}

	info, err := app.fetchProviderInfo(r.Context(), provider, cfg, token)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if info.id == "" {
		app.badRequestResponse(w, r, errors.New("provider returned no user id"))
		return
	}

	userID, err := app.findOrCreateOAuthUser(r.Context(), provider, info)
	if err != nil {
		switch err {
		case errNoVerifiedEmail, store.ErrConflict:
			app.badRequestResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.issueSession(r.Context(), w, userID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	dest := app.config.auth.dashboardURL + "/dashboard/"
	if c, err := r.Cookie(oauthRedirectCookie); err == nil && strings.HasPrefix(c.Value, "/") {
		dest = app.config.auth.dashboardURL + c.Value
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthRedirectCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   app.config.auth.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, dest, http.StatusTemporaryRedirect)
}

func (app *application) fetchProviderInfo(ctx context.Context, provider string, cfg *oauth2.Config, token *oauth2.Token) (*providerInfo, error) {
	client := cfg.Client(ctx, token)
	switch provider {
	case "google":
		return fetchGoogleInfo(ctx, client)
	case "github":
		return fetchGitHubInfo(ctx, client)
	default:
		return nil, fmt.Errorf("unknown provider %q", provider)
	}
}

func fetchGoogleInfo(ctx context.Context, client *http.Client) (*providerInfo, error) {
	var body struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := getJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &body); err != nil {
		return nil, err
	}
	return &providerInfo{id: body.Sub, email: body.Email, emailVerified: body.EmailVerified, name: body.Name}, nil
}

func fetchGitHubInfo(ctx context.Context, client *http.Client) (*providerInfo, error) {
	var user struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user", &user); err != nil {
		return nil, err
	}

	// GitHub's primary email needs a separate, scoped call; pick verified primary.
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
		return nil, err
	}
	info := &providerInfo{id: fmt.Sprintf("%d", user.ID), name: user.Name}
	if info.name == "" {
		info.name = user.Login
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			info.email = e.Email
			info.emailVerified = true
			break
		}
	}
	return info, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("userinfo %s: status %d: %s", url, resp.StatusCode, b)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

var errNoVerifiedEmail = errors.New("provider did not return a verified email")

func (app *application) findOrCreateOAuthUser(ctx context.Context, provider string, info *providerInfo) (userID uuid.UUID, err error) {
	id, err := app.store.UserIdentities.GetByProvider(ctx, provider, info.id)
	if err == nil {
		return id.UserID, nil
	}
	switch err {
	case store.ErrNotFound:
	default:
		return uuid.Nil, err
	}

	if info.email == "" || !info.emailVerified {
		return uuid.Nil, errNoVerifiedEmail
	}

	u, err := app.store.Users.GetByEmail(ctx, info.email)
	if err == nil {
		if linkErr := app.store.UserIdentities.Create(ctx, &store.UserIdentity{
			UserID: u.ID, Provider: provider, ProviderUserID: info.id, Email: info.email,
		}); linkErr != nil {
			switch linkErr {
			case store.ErrConflict:
			default:
				return uuid.Nil, linkErr
			}
		}
		return u.ID, nil
	}
	switch err {
	case store.ErrNotFound:
	default:
		return uuid.Nil, err
	}

	// Provider already verified the email, so mark it verified immediately.
	newUser := &store.User{Email: info.email, Name: info.name, IsVerified: true}
	newIdentity := &store.UserIdentity{Provider: provider, ProviderUserID: info.id, Email: info.email}
	if err := app.store.Users.CreateWithIdentity(ctx, newUser, newIdentity); err != nil {
		switch err {
		case store.ErrConflict:
			existing, lookupErr := app.store.Users.GetByEmail(ctx, info.email)
			if lookupErr != nil {
				return uuid.Nil, lookupErr
			}
			if linkErr := app.store.UserIdentities.Create(ctx, &store.UserIdentity{
				UserID: existing.ID, Provider: provider, ProviderUserID: info.id, Email: info.email,
			}); linkErr != nil {
				switch linkErr {
				case store.ErrConflict:
				default:
					return uuid.Nil, linkErr
				}
			}
			return existing.ID, nil
		default:
			return uuid.Nil, err
		}
	}
	return newUser.ID, nil
}
