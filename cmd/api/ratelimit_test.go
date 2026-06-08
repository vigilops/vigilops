package main

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestKeyFunc_returnsEmptyStringWhenMissing(t *testing.T) {
	app := &application{}
	r := httptest.NewRequest("POST", "/", nil)
	key, err := app.ingestKeyFunc(r)
	require.NoError(t, err)
	assert.Equal(t, "", key, "missing api_key_id must fail open as empty string")
}

func TestIngestKeyFunc_returnsKeyIDStringWhenSet(t *testing.T) {
	id := uuid.New()
	app := &application{}
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(withAPIKeyID(r.Context(), id))

	key, err := app.ingestKeyFunc(r)
	require.NoError(t, err)
	assert.Equal(t, id.String(), key)
}
