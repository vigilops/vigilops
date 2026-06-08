package main

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAtWindow_acceptsZSuffix(t *testing.T) {
	r := httptest.NewRequest("GET", "/?at=2026-06-08T04:30:00Z", nil)
	from, to, err := parseAtWindow(r)
	require.NoError(t, err)
	assert.True(t, from.Before(to))
}

func TestParseAtWindow_acceptsTimezoneOffset(t *testing.T) {
	r := httptest.NewRequest("GET", "/?at=2026-06-08T12:30:00%2B08:00", nil)
	from, to, err := parseAtWindow(r)
	require.NoError(t, err)
	assert.True(t, from.Before(to))
}

func TestParseAtWindow_recoversUrlDecodedPlus(t *testing.T) {
	// Caller forgot to URL-encode + → query parser hands us a space instead.
	// Real-world common when humans curl with the raw timestamp from a prior
	// response body.
	r := httptest.NewRequest("GET", "/?at=2026-06-08T12:30:00+08:00", nil)
	from, to, err := parseAtWindow(r)
	require.NoError(t, err, "should recover the lost + in the timezone offset")
	assert.True(t, from.Before(to))
}

func TestParseAtWindow_defaultsTo30DayWindowWhenEmpty(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	from, to, err := parseAtWindow(r)
	require.NoError(t, err)
	assert.InDelta(t, defaultWindow.Hours(), to.Sub(from).Hours(), 0.01)
}

func TestParseAtWindow_rejectsGarbage(t *testing.T) {
	r := httptest.NewRequest("GET", "/?at=not-a-timestamp", nil)
	_, _, err := parseAtWindow(r)
	assert.Error(t, err)
}
