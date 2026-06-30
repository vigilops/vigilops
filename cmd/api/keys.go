package main

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/keelwave/keelwave/internal/auth"
	"github.com/keelwave/keelwave/internal/store"
)

type createKeyPayload struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type createKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateKey godoc
//
//	@Summary		Creates an API key
//	@Description	Generates a new API key for a project. Plaintext key is returned ONCE. Requires admin role.
//	@Tags			admin/keys
//	@Accept			json
//	@Produce		json
//	@Param			orgID		path		string				true	"Organization UUID"
//	@Param			projectID	path		string				true	"Project UUID"
//	@Param			payload		body		createKeyPayload	true	"Key payload"
//	@Success		201			{object}	createKeyResponse
//	@Failure		400			{object}	error
//	@Failure		404			{object}	error
//	@Failure		500			{object}	error
//	@Router			/admin/orgs/{orgID}/projects/{projectID}/keys [post]
func (app *application) createKeyHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	projectID, err := parseUUIDParam(r, "projectID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload createKeyPayload
	if err = readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err = Validate.Struct(payload); err != nil {
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
	if proj.OrganizationID != orgID {
		app.notFoundResponse(w, r, store.ErrNotFound)
		return
	}

	plaintext, hash, err := auth.Generate()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	k := &store.APIKey{
		ProjectID: projectID,
		KeyHash:   hash,
		Name:      payload.Name,
	}
	if err := app.store.APIKeys.Create(r.Context(), k); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	resp := createKeyResponse{
		ID:        k.ID,
		ProjectID: k.ProjectID,
		Name:      k.Name,
		Key:       plaintext,
		CreatedAt: k.CreatedAt,
	}
	if err := app.jsonResponse(w, http.StatusCreated, resp); err != nil {
		app.internalServerError(w, r, err)
	}
}

// ListKeys godoc
//
//	@Summary	Lists API keys for a project
//	@Tags		admin/keys
//	@Produce	json
//	@Param		orgID		path		string	true	"Organization UUID"
//	@Param		projectID	path		string	true	"Project UUID"
//	@Success	200			{array}		store.APIKey
//	@Failure	400			{object}	error
//	@Failure	404			{object}	error
//	@Failure	500			{object}	error
//	@Router		/admin/orgs/{orgID}/projects/{projectID}/keys [get]
func (app *application) listKeysHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
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
	if proj.OrganizationID != orgID {
		app.notFoundResponse(w, r, store.ErrNotFound)
		return
	}

	keys, err := app.store.APIKeys.ListByProject(r.Context(), projectID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, keys); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteKey godoc
//
//	@Summary		Deletes an API key
//	@Description	Revokes an API key. Subsequent requests using this key will return 401. Requires admin role.
//	@Tags			admin/keys
//	@Param			orgID		path	string	true	"Organization UUID"
//	@Param			projectID	path	string	true	"Project UUID"
//	@Param			keyID		path	string	true	"Key UUID"
//	@Success		204
//	@Failure		400	{object}	error
//	@Failure		404	{object}	error
//	@Failure		500	{object}	error
//	@Router			/admin/orgs/{orgID}/projects/{projectID}/keys/{keyID} [delete]
func (app *application) deleteKeyHandler(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUIDParam(r, "orgID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	projectID, err := parseUUIDParam(r, "projectID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	keyID, err := parseUUIDParam(r, "keyID")
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
	if proj.OrganizationID != orgID {
		app.notFoundResponse(w, r, store.ErrNotFound)
		return
	}

	if err := app.store.APIKeys.Delete(r.Context(), keyID, projectID); err != nil {
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
