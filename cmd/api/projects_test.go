package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProjectHandler_creates201WithName(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	var body struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodPost, srv.URL+"/v1/admin/projects", "", map[string]any{"name": "newproj"}, &body)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "body=%s", raw)
	assert.Equal(t, "newproj", body.Data.Name)
	assert.NotEmpty(t, body.Data.ID)
}

func TestCreateProjectHandler_rejectsMissingName(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/v1/admin/projects", "", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestListProjectsHandler_includesSeeded(t *testing.T) {
	srv, projectID, _, _ := newTestServer(t)
	var body struct {
		Data []struct{ ID string } `json:"data"`
	}
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/admin/projects", "", nil, &body)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var found bool
	for _, p := range body.Data {
		if p.ID == projectID {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGetProjectHandler_returnsRowOrNotFound(t *testing.T) {
	srv, projectID, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/admin/projects/"+projectID, "", nil, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp2, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/admin/projects/00000000-0000-0000-0000-000000000000", "", nil, nil)
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

func TestGetProjectHandler_rejectsBadUUID(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/admin/projects/not-a-uuid", "", nil, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteProjectHandler_returnsNoContent(t *testing.T) {
	srv, projectID, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodDelete, srv.URL+"/v1/admin/projects/"+projectID, "", nil, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
