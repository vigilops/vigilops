package main

import (
	"net/http"

	"github.com/keelwave/keelwave/internal/store"
)

type updateOrgPayload struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

// UpdateOrg godoc
//
//	@Summary	Rename an organization
//	@Tags		admin/orgs
//	@Accept		json
//	@Produce	json
//	@Param		orgID	path		string				true	"Organization UUID"
//	@Param		payload	body		updateOrgPayload	true	"Update payload"
//	@Success	200		{object}	store.Organization
//	@Failure	400		{object}	error
//	@Failure	403		{object}	error
//	@Failure	404		{object}	error
//	@Router		/admin/orgs/{orgID} [patch]
func (app *application) updateOrgHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload updateOrgPayload
	if err = readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err = Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	org, err := app.store.Organizations.Update(r.Context(), orgID, payload.Name)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, org); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteOrg godoc
//
//	@Summary		Delete an organization
//	@Description	Permanently deletes the org and all its projects, keys, and ingest data.
//	@Tags			admin/orgs
//	@Param			orgID	path	string	true	"Organization UUID"
//	@Success		204
//	@Failure		400	{object}	error
//	@Failure		403	{object}	error
//	@Failure		404	{object}	error
//	@Router			/admin/orgs/{orgID} [delete]
func (app *application) deleteOrgHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.store.Organizations.Delete(r.Context(), orgID); err != nil {
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
