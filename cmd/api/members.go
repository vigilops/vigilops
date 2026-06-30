package main

import (
	"errors"
	"net/http"

	"github.com/keelwave/keelwave/internal/store"
)

type updateRolePayload struct {
	Role string `json:"role" validate:"required,oneof=admin member"`
}

// ListMembers godoc
//
//	@Summary	List org members
//	@Tags		admin/members
//	@Produce	json
//	@Param		orgID	path		string	true	"Organization UUID"
//	@Success	200		{array}		store.MemberWithUser
//	@Failure	403		{object}	error
//	@Router		/admin/orgs/{orgID}/members [get]
func (app *application) listMembersHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	members, err := app.store.OrganizationMembers.ListByOrganization(r.Context(), orgID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, members); err != nil {
		app.internalServerError(w, r, err)
	}
}

// UpdateMemberRole godoc
//
//	@Summary		Update a member's role
//	@Description	Owner only. Cannot change an owner's role or promote to owner.
//	@Tags			admin/members
//	@Accept			json
//	@Produce		json
//	@Param			orgID	path	string				true	"Organization UUID"
//	@Param			userID	path	string				true	"Member user UUID"
//	@Param			payload	body	updateRolePayload	true	"Role payload"
//	@Success		204
//	@Failure		400	{object}	error
//	@Failure		403	{object}	error
//	@Failure		404	{object}	error
//	@Router			/admin/orgs/{orgID}/members/{userID}/role [patch]
func (app *application) updateMemberRoleHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	targetUserID, err := parseUUIDParam(r, "userID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload updateRolePayload
	if err = readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err = Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	target, err := app.store.OrganizationMembers.Get(r.Context(), orgID, targetUserID)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if target.Role == store.OwnerRole {
		app.forbiddenResponse(w, r)
		return
	}

	if err := app.store.OrganizationMembers.UpdateRole(r.Context(), orgID, targetUserID, payload.Role); err != nil {
		switch err {
		case store.ErrNotFound:
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.noContentResponse(w)
}

// RemoveMember godoc
//
//	@Summary		Remove a member from the org
//	@Description	Admin+. Cannot remove an owner or yourself.
//	@Tags			admin/members
//	@Param			orgID	path	string	true	"Organization UUID"
//	@Param			userID	path	string	true	"Member user UUID"
//	@Success		204
//	@Failure		400	{object}	error
//	@Failure		403	{object}	error
//	@Failure		404	{object}	error
//	@Router			/admin/orgs/{orgID}/members/{userID} [delete]
func (app *application) removeMemberHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	targetUserID, err := parseUUIDParam(r, "userID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	callerID := userIDFromContext(r.Context())
	if callerID == targetUserID {
		app.conflictResponse(w, r, errors.New("cannot remove yourself"))
		return
	}

	target, err := app.store.OrganizationMembers.Get(r.Context(), orgID, targetUserID)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	if target.Role == store.OwnerRole {
		app.forbiddenResponse(w, r)
		return
	}

	if err := app.store.OrganizationMembers.Remove(r.Context(), orgID, targetUserID); err != nil {
		switch err {
		case store.ErrNotFound:
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.noContentResponse(w)
}
