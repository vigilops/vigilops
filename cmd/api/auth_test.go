package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/keelwave/keelwave/internal/auth"
	"github.com/keelwave/keelwave/internal/store"
)

func TestRegisterHandler_creates201(t *testing.T) {
	ts := newTestServer(t)

	email := fmt.Sprintf("reg-%d@dev.local", time.Now().UnixNano())
	var body map[string]any
	resp, _ := doJSON(t, http.MethodPost, ts.srv.URL+"/v1/auth/register", "",
		map[string]any{"email": email, "password": "password123", "name": "Test User"},
		&body)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	data := body["data"].(map[string]any)
	user := data["user"].(map[string]any)
	assert.Equal(t, email, user["email"])
	assert.False(t, user["is_verified"].(bool))
	assert.Empty(t, data["organizations"])

	// cookie issued
	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == testCookieName {
			found = true
		}
	}
	assert.True(t, found, "session cookie should be set")
}

func TestRegisterHandler_conflictOnDuplicateEmail(t *testing.T) {
	ts := newTestServer(t)

	email := fmt.Sprintf("dup-%d@dev.local", time.Now().UnixNano())
	payload := map[string]any{"email": email, "password": "password123"}

	resp, _ := doJSON(t, http.MethodPost, ts.srv.URL+"/v1/auth/register", "", payload, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	resp2, _ := doJSON(t, http.MethodPost, ts.srv.URL+"/v1/auth/register", "", payload, nil)
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestVerifyEmailHandler_204OnValidToken(t *testing.T) {
	ts := newTestServer(t)
	ctx := context.Background()
	s := store.NewStorage(testPool)

	// create a user + verification token
	u := &store.User{Email: fmt.Sprintf("verify-%d@dev.local", time.Now().UnixNano()), Name: "v"}
	require.NoError(t, u.Password.Set("pass1234"))
	require.NoError(t, s.Users.Create(ctx, u, nil))

	plaintext, hash, err := auth.GenerateSession()
	require.NoError(t, err)
	v := &store.UserVerification{UserID: u.ID, Token: hash, ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, s.UserVerifications.Create(ctx, v))

	resp, _ := doJSON(t, http.MethodGet, ts.srv.URL+"/v1/auth/verify-email/"+plaintext, "", nil, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// user should now be verified
	updated, err := s.Users.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, updated.IsVerified)
	assert.NotNil(t, updated.VerifiedAt)
}

func TestVerifyEmailHandler_404OnExpiredToken(t *testing.T) {
	ts := newTestServer(t)
	ctx := context.Background()
	s := store.NewStorage(testPool)

	u := &store.User{Email: fmt.Sprintf("exp-%d@dev.local", time.Now().UnixNano()), Name: "e"}
	require.NoError(t, u.Password.Set("pass1234"))
	require.NoError(t, s.Users.Create(ctx, u, nil))

	plaintext, hash, err := auth.GenerateSession()
	require.NoError(t, err)
	v := &store.UserVerification{UserID: u.ID, Token: hash, ExpiresAt: time.Now().Add(-time.Hour)}
	require.NoError(t, s.UserVerifications.Create(ctx, v))

	resp, _ := doJSON(t, http.MethodGet, ts.srv.URL+"/v1/auth/verify-email/"+plaintext, "", nil, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestVerifyEmailHandler_404OnBadToken(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, ts.srv.URL+"/v1/auth/verify-email/notavalidtoken", "", nil, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateOrgHandler_201(t *testing.T) {
	ts := newTestServer(t)

	var body map[string]any
	resp, _ := doJSON(t, http.MethodPost, ts.srv.URL+"/v1/admin/orgs", "",
		map[string]any{"name": "my-new-org"},
		&body, ts.cookie)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	data := body["data"].(map[string]any)
	assert.Equal(t, "my-new-org", data["name"])
	assert.NotEmpty(t, data["id"])
}

func TestCreateOrgHandler_401WithoutSession(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, ts.srv.URL+"/v1/admin/orgs", "",
		map[string]any{"name": "x"}, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
