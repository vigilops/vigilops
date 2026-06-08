package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAITraceStore_Insert_assignsIDAndTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "ai")

	in, out, tot := 100, 50, 150
	cost := 0.0023
	provider := "anthropic"

	tr := &AITrace{
		ProjectID:    p.ID,
		Model:        "claude-opus-4-7",
		Provider:     &provider,
		InputTokens:  &in,
		OutputTokens: &out,
		TotalTokens:  &tot,
		CostUSD:      &cost,
		Status:       "success",
		Metadata:     []byte(`{"k":"v"}`),
	}
	require.NoError(t, s.AITraces.Insert(ctx, tr))
	assert.NotEqual(t, uuid.Nil, tr.ID)
	assert.False(t, tr.Timestamp.IsZero())
}
