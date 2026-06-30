package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProjectHandler_creates201WithName(t *testing.T) {
	ts := newTestServer(t)
	var body struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	url := ts.srv.URL + "/v1/admin/orgs/" + ts.orgID + "/projects"
	resp, raw := doJSON(t, http.MethodPost, url, "", map[string]any{"name": "newproj"}, &body, ts.cookie)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "body=%s", raw)
	assert.Equal(t, "newproj", body.Data.Name)
	assert.NotEmpty(t, body.Data.ID)
}

func TestCreateProjectHandler_rejectsMissingName(t *testing.T) {
	ts := newTestServer(t)
	url := ts.srv.URL + "/v1/admin/orgs/" + ts.orgID + "/projects"
	resp, _ := doJSON(t, http.MethodPost, url, "", map[string]any{}, nil, ts.cookie)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestListProjectsHandler_includesSeeded(t *testing.T) {
	ts := newTestServer(t)
	var body struct {
		Data []struct{ ID string } `json:"data"`
	}
	url := ts.srv.URL + "/v1/admin/orgs/" + ts.orgID + "/projects"
	resp, _ := doJSON(t, http.MethodGet, url, "", nil, &body, ts.cookie)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var found bool
	for _, p := range body.Data {
		if p.ID == ts.projID {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGetProjectHandler_returnsRowOrNotFound(t *testing.T) {
	ts := newTestServer(t)
	base := ts.srv.URL + "/v1/admin/orgs/" + ts.orgID + "/projects/"
	resp, _ := doJSON(t, http.MethodGet, base+ts.projID, "", nil, nil, ts.cookie)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp2, _ := doJSON(t, http.MethodGet, base+"00000000-0000-0000-0000-000000000000", "", nil, nil, ts.cookie)
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

func TestGetProjectHandler_rejectsBadUUID(t *testing.T) {
	ts := newTestServer(t)
	url := ts.srv.URL + "/v1/admin/orgs/" + ts.orgID + "/projects/not-a-uuid"
	resp, _ := doJSON(t, http.MethodGet, url, "", nil, nil, ts.cookie)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteProjectHandler_returnsNoContent(t *testing.T) {
	ts := newTestServer(t)
	url := ts.srv.URL + "/v1/admin/orgs/" + ts.orgID + "/projects/" + ts.projID
	resp, _ := doJSON(t, http.MethodDelete, url, "", nil, nil, ts.cookie)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
