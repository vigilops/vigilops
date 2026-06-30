package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/keelwave/keelwave/internal/auth"
	"github.com/keelwave/keelwave/internal/mailer"
	"github.com/keelwave/keelwave/internal/store"
)

type createInvitePayload struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role"  validate:"omitempty,oneof=admin member"`
}

// CreateInvite godoc
//
//	@Summary	Invite a user to an organization
//	@Tags		admin/invites
//	@Accept		json
//	@Produce	json
//	@Param		orgID	path		string				true	"Organization UUID"
//	@Param		payload	body		createInvitePayload	true	"Invite payload"
//	@Success	201		{object}	store.OrganizationInvite
//	@Failure	400		{object}	error
//	@Failure	403		{object}	error
//	@Router		/admin/orgs/{orgID}/invites [post]
func (app *application) createInviteHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload createInvitePayload
	if err = readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err = Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if payload.Role == "" {
		payload.Role = "member"
	}

	plaintext, hash, err := auth.GenerateSession()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	inv := &store.OrganizationInvite{
		OrganizationID: orgID,
		Email:          payload.Email,
		Role:           payload.Role,
		Token:          hash,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
	}
	if err := app.store.OrganizationInvites.Create(r.Context(), inv); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	type response struct {
		store.OrganizationInvite
		InviteURL string `json:"invite_url"`
	}
	inviteURL := app.config.auth.dashboardURL + "/invite/" + plaintext
	out := response{
		OrganizationInvite: *inv,
		InviteURL:          inviteURL,
	}

	org, err := app.store.Organizations.GetByID(r.Context(), orgID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	go func() {
		data := struct {
			OrgName   string
			Role      string
			InviteURL string
		}{OrgName: org.Name, Role: inv.Role, InviteURL: inviteURL}
		if err := app.mailer.Send(mailer.InviteTemplate, inv.Email, data); err != nil {
			app.logger.Warnw("invite email failed", "to", inv.Email, "err", err)
		}
	}()

	if err := app.jsonResponse(w, http.StatusCreated, out); err != nil {
		app.internalServerError(w, r, err)
	}
}

// ListInvites godoc
//
//	@Summary	List pending invites for an organization
//	@Tags		admin/invites
//	@Produce	json
//	@Param		orgID	path	string	true	"Organization UUID"
//	@Success	200		{array}	store.OrganizationInvite
//	@Router		/admin/orgs/{orgID}/invites [get]
func (app *application) listInvitesHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	invites, err := app.store.OrganizationInvites.ListByOrg(r.Context(), orgID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, invites); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteInvite godoc
//
//	@Summary	Cancel an invite
//	@Tags		admin/invites
//	@Param		orgID		path	string	true	"Organization UUID"
//	@Param		inviteID	path	string	true	"Invite UUID"
//	@Success	204
//	@Router		/admin/orgs/{orgID}/invites/{inviteID} [delete]
func (app *application) deleteInviteHandler(w http.ResponseWriter, r *http.Request) {
	inviteID, err := parseUUIDParam(r, "inviteID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.store.OrganizationInvites.Delete(r.Context(), inviteID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			app.notFoundResponse(w, r, err)
			return
		}
		app.internalServerError(w, r, err)
		return
	}
	app.noContentResponse(w)
}

// GetInvite godoc
//
//	@Summary	Get invite info by token (public — for the accept page)
//	@Tags		auth
//	@Produce	json
//	@Param		token	path		string	true	"Invite token"
//	@Success	200		{object}	store.OrganizationInvite
//	@Router		/auth/invites/{token} [get]
func (app *application) getInviteHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		app.badRequestResponse(w, r, errors.New("token required"))
		return
	}

	hash := auth.HashSession(token)
	inv, err := app.store.OrganizationInvites.GetByToken(r.Context(), hash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			app.notFoundResponse(w, r, err)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if time.Now().After(inv.ExpiresAt) {
		app.notFoundResponse(w, r, errors.New("invite expired"))
		return
	}
	if inv.AcceptedAt != nil {
		app.notFoundResponse(w, r, errors.New("invite already accepted"))
		return
	}

	org, err := app.store.Organizations.GetByID(r.Context(), inv.OrganizationID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	type response struct {
		OrgName string `json:"org_name"`
		Role    string `json:"role"`
	}
	if err := app.jsonResponse(w, http.StatusOK, response{
		OrgName: org.Name,
		Role:    inv.Role,
	}); err != nil {
		app.internalServerError(w, r, err)
	}
}

// AcceptInvite godoc
//
//	@Summary	Accept an invite (requires session)
//	@Tags		auth
//	@Param		token	path	string	true	"Invite token"
//	@Success	204
//	@Router		/auth/invites/{token}/accept [put]
func (app *application) acceptInviteHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		app.badRequestResponse(w, r, errors.New("token required"))
		return
	}

	hash := auth.HashSession(token)
	inv, err := app.store.OrganizationInvites.GetByToken(r.Context(), hash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			app.notFoundResponse(w, r, err)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if time.Now().After(inv.ExpiresAt) {
		app.badRequestResponse(w, r, errors.New("invite expired"))
		return
	}
	if inv.AcceptedAt != nil {
		app.badRequestResponse(w, r, errors.New("invite already accepted"))
		return
	}

	userID := userIDFromContext(r.Context())
	user, err := app.store.Users.GetByID(r.Context(), userID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if !strings.EqualFold(user.Email, inv.Email) {
		app.forbiddenResponse(w, r)
		return
	}

	if err := app.store.OrganizationInvites.Accept(r.Context(), inv.ID, userID, inv.Role); err != nil {
		if errors.Is(err, store.ErrConflict) {
			app.noContentResponse(w)
			return
		}
		app.internalServerError(w, r, err)
		return
	}
	app.noContentResponse(w)
}
