package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateKeyHandler_returnsPlaintextOnce(t *testing.T) {
	ts := newTestServer(t)
	var body struct {
		Data struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodPost,
		ts.srv.URL+"/v1/admin/orgs/"+ts.orgID+"/projects/"+ts.projID+"/keys", "",
		map[string]any{"name": "fresh"}, &body, ts.cookie)

	require.Equal(t, http.StatusCreated, resp.StatusCode, "body=%s", raw)
	assert.True(t, strings.HasPrefix(body.Data.Key, "kw_"), "plaintext returned with prefix")
	assert.NotEmpty(t, body.Data.ID)
}

func TestCreateKeyHandler_404WhenProjectMissing(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost,
		ts.srv.URL+"/v1/admin/orgs/"+ts.orgID+"/projects/00000000-0000-0000-0000-000000000000/keys", "",
		map[string]any{"name": "fresh"}, nil, ts.cookie)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestListKeysHandler_omitsKeyHash(t *testing.T) {
	ts := newTestServer(t)
	keysURL := ts.srv.URL + "/v1/admin/orgs/" + ts.orgID + "/projects/" + ts.projID + "/keys"
	doJSON(t, http.MethodPost, keysURL, "", map[string]any{"name": "k1"}, nil, ts.cookie)

	resp, raw := doJSON(t, http.MethodGet, keysURL, "", nil, nil, ts.cookie)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotContains(t, string(raw), "key_hash", "raw bytea must never leave the server")
	assert.NotContains(t, string(raw), `"key"`, "plaintext is returned only on create")
}

func TestDeleteKeyHandler_204ThenNotFound(t *testing.T) {
	ts := newTestServer(t)

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	doJSON(t, http.MethodPost, ts.srv.URL+"/v1/admin/orgs/"+ts.orgID+"/projects/"+ts.projID+"/keys", "",
		map[string]any{"name": "k"}, &created, ts.cookie)

	keyURL := ts.srv.URL + "/v1/admin/orgs/" + ts.orgID + "/projects/" + ts.projID + "/keys/" + created.Data.ID
	resp, _ := doJSON(t, http.MethodDelete, keyURL, "", nil, nil, ts.cookie)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp2, _ := doJSON(t, http.MethodDelete, keyURL, "", nil, nil, ts.cookie)
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}
