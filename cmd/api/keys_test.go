package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateKeyHandler_returnsPlaintextOnce(t *testing.T) {
	srv, projectID, _, _ := newTestServer(t)

	var body struct {
		Data struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodPost,
		srv.URL+"/v1/admin/projects/"+projectID+"/keys", "",
		map[string]any{"name": "fresh"}, &body)

	require.Equal(t, http.StatusCreated, resp.StatusCode, "body=%s", raw)
	assert.True(t, strings.HasPrefix(body.Data.Key, "kw_"), "plaintext returned with prefix")
	assert.NotEmpty(t, body.Data.ID)
}

func TestCreateKeyHandler_404WhenProjectMissing(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost,
		srv.URL+"/v1/admin/projects/00000000-0000-0000-0000-000000000000/keys", "",
		map[string]any{"name": "fresh"}, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestListKeysHandler_omitsKeyHash(t *testing.T) {
	srv, projectID, _, _ := newTestServer(t)
	doJSON(t, http.MethodPost, srv.URL+"/v1/admin/projects/"+projectID+"/keys", "",
		map[string]any{"name": "k1"}, nil)

	resp, raw := doJSON(t, http.MethodGet, srv.URL+"/v1/admin/projects/"+projectID+"/keys", "", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotContains(t, string(raw), "key_hash", "raw bytea must never leave the server")
	assert.NotContains(t, string(raw), `"key"`, "plaintext is returned only on create")
}

func TestDeleteKeyHandler_204ThenNotFound(t *testing.T) {
	srv, projectID, _, _ := newTestServer(t)

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	doJSON(t, http.MethodPost, srv.URL+"/v1/admin/projects/"+projectID+"/keys", "",
		map[string]any{"name": "k"}, &created)

	resp, _ := doJSON(t, http.MethodDelete, srv.URL+"/v1/admin/keys/"+created.Data.ID, "", nil, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp2, _ := doJSON(t, http.MethodDelete, srv.URL+"/v1/admin/keys/"+created.Data.ID, "", nil, nil)
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}
