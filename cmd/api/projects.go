package main

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/keelwave/keelwave/internal/store"
)

type createProjectPayload struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

// CreateProject godoc
//
//	@Summary		Creates a project
//	@Description	Creates a tenant project; all ingest rows are scoped by project_id.
//	@Tags			admin/projects
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		createProjectPayload	true	"Project payload"
//	@Success		201		{object}	store.Project
//	@Failure		400		{object}	error
//	@Failure		500		{object}	error
//	@Router			/admin/projects [post]
func (app *application) createProjectHandler(w http.ResponseWriter, r *http.Request) {
	var payload createProjectPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	p := &store.Project{Name: payload.Name}
	if err := app.store.Projects.Create(r.Context(), p); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, p); err != nil {
		app.internalServerError(w, r, err)
	}
}

// ListProjects godoc
//
//	@Summary		Lists projects
//	@Description	Returns all projects ordered by created_at desc.
//	@Tags			admin/projects
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		store.Project
//	@Failure		500	{object}	error
//	@Router			/admin/projects [get]
func (app *application) listProjectsHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := app.store.Projects.List(r.Context())
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, projects); err != nil {
		app.internalServerError(w, r, err)
	}
}

// GetProject godoc
//
//	@Summary		Fetches a project
//	@Description	Fetches a project by UUID.
//	@Tags			admin/projects
//	@Accept			json
//	@Produce		json
//	@Param			projectID	path		string	true	"Project UUID"
//	@Success		200			{object}	store.Project
//	@Failure		400			{object}	error
//	@Failure		404			{object}	error
//	@Failure		500			{object}	error
//	@Router			/admin/projects/{projectID} [get]
func (app *application) getProjectHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "projectID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	p, err := app.store.Projects.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, p); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteProject godoc
//
//	@Summary		Deletes a project
//	@Description	Deletes a project and cascades to its api_keys and ingest rows.
//	@Tags			admin/projects
//	@Accept			json
//	@Produce		json
//	@Param			projectID	path	string	true	"Project UUID"
//	@Success		204
//	@Failure		400	{object}	error
//	@Failure		404	{object}	error
//	@Failure		500	{object}	error
//	@Router			/admin/projects/{projectID} [delete]
func (app *application) deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "projectID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.store.Projects.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseUUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
}
