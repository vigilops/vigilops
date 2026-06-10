package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentToolStore_UpsertSeen_insertsThenBumpsLastSeen(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "tools")

	require.NoError(t, s.AgentTools.UpsertSeen(ctx, p.ID, "search"))
	first, err := s.AgentTools.ListByProject(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, first, 1)
	firstSeen := first[0].FirstSeenAt
	firstLast := first[0].LastSeenAt

	require.NoError(t, s.AgentTools.UpsertSeen(ctx, p.ID, "search"))
	second, err := s.AgentTools.ListByProject(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, second, 1, "ON CONFLICT should not insert a duplicate")

	assert.True(t, firstSeen.Equal(second[0].FirstSeenAt), "first_seen_at preserved")
	assert.True(t, second[0].LastSeenAt.After(firstLast) || second[0].LastSeenAt.Equal(firstLast), "last_seen_at bumped")
}

func TestAgentToolStore_ListByProject_returnsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "tools-empty")

	out, err := s.AgentTools.ListByProject(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Len(t, out, 0)
}

func TestAgentToolStore_ListByProject_isolatedAcrossProjects(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	a := testProject(t, s, "a")
	b := testProject(t, s, "b")

	require.NoError(t, s.AgentTools.UpsertSeen(ctx, a.ID, "search"))
	require.NoError(t, s.AgentTools.UpsertSeen(ctx, b.ID, "calculator"))

	listA, _ := s.AgentTools.ListByProject(ctx, a.ID)
	listB, _ := s.AgentTools.ListByProject(ctx, b.ID)

	require.Len(t, listA, 1)
	require.Len(t, listB, 1)
	assert.Equal(t, "search", listA[0].ToolName)
	assert.Equal(t, "calculator", listB[0].ToolName)
}
