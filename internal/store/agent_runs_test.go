package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRunStore_GetByID_returnsRowForProject(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "getbyid")

	input := "find X"
	run := &AgentRun{
		ProjectID: p.ID,
		AgentName: "researcher",
		Status:    "running",
		Input:     &input,
	}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	from := run.Timestamp.Add(-time.Hour)
	to := run.Timestamp.Add(time.Hour)
	got, err := s.AgentRuns.GetByID(ctx, p.ID, run.ID, from, to)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, run.ID, got.ID)
	assert.Equal(t, p.ID, got.ProjectID)
	assert.Equal(t, "researcher", got.AgentName)
	assert.Equal(t, "running", got.Status)
	require.NotNil(t, got.Input)
	assert.Equal(t, "find X", *got.Input)
}

func TestAgentRunStore_GetByID_returnsNotFoundForOtherProject(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	owner := testProject(t, s, "owner")
	other := testProject(t, s, "other")

	run := &AgentRun{ProjectID: owner.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	from := run.Timestamp.Add(-time.Hour)
	to := run.Timestamp.Add(time.Hour)
	got, err := s.AgentRuns.GetByID(ctx, other.ID, run.ID, from, to)

	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, got)
}

func TestAgentRunStore_Insert_assignsIDAndDefaultTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "runinsert")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	assert.NotEqual(t, uuid.Nil, run.ID)
	assert.False(t, run.Timestamp.IsZero())
}

func TestAgentRunStore_Finish_setsTerminalFields(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "runfinish")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	out := "answer"
	reason := "clean"
	require.NoError(t, s.AgentRuns.Finish(ctx, run.ID, run.Timestamp, AgentRunFinish{
		Status:            "completed",
		TerminationReason: &reason,
		TotalSteps:        3,
		TotalTokens:       100,
		Output:            &out,
	}))

	got, err := s.AgentRuns.GetByID(ctx, p.ID, run.ID, run.Timestamp.Add(-time.Second), run.Timestamp.Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, "completed", got.Status)
	require.NotNil(t, got.TerminationReason)
	assert.Equal(t, "clean", *got.TerminationReason)
	assert.Equal(t, 3, got.TotalSteps)
	assert.Equal(t, 100, got.TotalTokens)
	require.NotNil(t, got.FinishedAt)
}

func TestAgentRunStore_Finish_notFoundOnWrongTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "runfinish-miss")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	err := s.AgentRuns.Finish(ctx, run.ID, time.Now().Add(time.Hour), AgentRunFinish{Status: "completed"})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestAgentRunStore_ListByProject_paginatesNewestFirst(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "list")

	now := time.Now()
	for i := 0; i < 5; i++ {
		run := &AgentRun{
			ProjectID: p.ID,
			AgentName: "a",
			Status:    "running",
			Timestamp: now.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, s.AgentRuns.Insert(ctx, run))
	}

	from := now.Add(-time.Hour)
	to := now.Add(time.Hour)

	first, err := s.AgentRuns.ListByProject(ctx, p.ID, from, to, 2, 0)
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.True(t, first[0].Timestamp.After(first[1].Timestamp), "newest first")

	second, err := s.AgentRuns.ListByProject(ctx, p.ID, from, to, 2, 2)
	require.NoError(t, err)
	require.Len(t, second, 2)
	assert.True(t, first[1].Timestamp.After(second[0].Timestamp), "page 2 older than page 1")
}
