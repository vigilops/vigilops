package store

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIEventStore_Insert_assignsIDAndTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "evt")

	ip := net.ParseIP("1.2.3.4")
	ua := "curl/8"

	e := &APIEvent{
		ProjectID:  p.ID,
		Service:    "api",
		Method:     "GET",
		Path:       "/users",
		StatusCode: 200,
		DurationMs: 12,
		IP:         &ip,
		UserAgent:  &ua,
	}
	require.NoError(t, s.APIEvents.Insert(ctx, e))
	assert.NotEqual(t, uuid.Nil, e.ID)
	assert.False(t, e.Timestamp.IsZero())
}
