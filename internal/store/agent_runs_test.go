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

func TestAgentRunStore_ListByProject_returnsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "runs-empty")

	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)
	runs, err := s.AgentRuns.ListByProject(ctx, p.ID, from, to, 10, 0)
	require.NoError(t, err)
	require.NotNil(t, runs)
	assert.Len(t, runs, 0)
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

func TestAgentRunStore_RunHealth_rollsUpPerAgentName(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "runhealth")

	// research-agent: 3 runs — 2 completed, 1 failed-with-loop
	// triage:         1 run  — completed
	mk := func(name, status string, loop bool, cost float64, tokens int) {
		run := &AgentRun{ProjectID: p.ID, AgentName: name, Status: "running"}
		require.NoError(t, s.AgentRuns.Insert(ctx, run))
		require.NoError(t, s.AgentRuns.Finish(ctx, run.ID, run.Timestamp, AgentRunFinish{
			Status:       status,
			LoopDetected: loop,
			TotalTokens:  tokens,
			TotalCostUSD: &cost,
		}))
	}
	mk("research-agent", "completed", false, 0.02, 1000)
	mk("research-agent", "completed", false, 0.04, 2000)
	mk("research-agent", "failed", true, 0.06, 3000)
	mk("triage", "completed", false, 0.01, 500)

	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)
	rows, err := s.AgentRuns.RunHealth(ctx, p.ID, from, to)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byName := map[string]*RunHealthRow{}
	for _, r := range rows {
		byName[r.AgentName] = r
	}

	ra := byName["research-agent"]
	require.NotNil(t, ra)
	assert.Equal(t, 3, ra.TotalRuns)
	assert.Equal(t, 2, ra.CompletedRuns)
	assert.Equal(t, 1, ra.LoopRuns)
	assert.InDelta(t, 2.0/3.0, ra.CompletionRate, 0.001)
	assert.InDelta(t, 1.0/3.0, ra.LoopRate, 0.001)
	require.NotNil(t, ra.AvgCostUSD)
	assert.InDelta(t, 0.04, *ra.AvgCostUSD, 0.001)
	assert.InDelta(t, 2000.0, ra.AvgTokens, 0.001)

	tr := byName["triage"]
	require.NotNil(t, tr)
	assert.Equal(t, 1, tr.TotalRuns)
	assert.InDelta(t, 1.0, tr.CompletionRate, 0.001)
}

func TestAgentRunStore_RunHealth_returnsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "runhealth-empty")

	rows, err := s.AgentRuns.RunHealth(ctx, p.ID,
		time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.NotNil(t, rows)
	assert.Len(t, rows, 0)
}

func TestAgentRunStore_RunsTimeseries_bucketsByInterval(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "timeseries")

	base := time.Now().Truncate(time.Hour).Add(-3 * time.Hour)
	// 3 runs in the same hour bucket: 2 completed, 1 failed+loop.
	mk := func(offsetMin int, status string, loop bool) {
		run := &AgentRun{
			ProjectID: p.ID,
			AgentName: "a",
			Status:    "running",
			Timestamp: base.Add(time.Duration(offsetMin) * time.Minute),
		}
		require.NoError(t, s.AgentRuns.Insert(ctx, run))
		require.NoError(t, s.AgentRuns.Finish(ctx, run.ID, run.Timestamp, AgentRunFinish{
			Status:       status,
			LoopDetected: loop,
		}))
	}
	mk(1, "completed", false)
	mk(5, "completed", false)
	mk(10, "failed", true)

	from := base.Add(-time.Hour)
	to := base.Add(time.Hour)
	buckets, err := s.AgentRuns.RunsTimeseries(ctx, p.ID, from, to, "1 hour")
	require.NoError(t, err)
	require.Len(t, buckets, 1, "all three runs fall in one hourly bucket")

	b := buckets[0]
	assert.Equal(t, 3, b.Total)
	assert.Equal(t, 2, b.Completed)
	assert.Equal(t, 1, b.Failed)
	assert.Equal(t, 1, b.Loop)
}

func TestAgentRunStore_RunsTimeseries_returnsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "timeseries-empty")

	buckets, err := s.AgentRuns.RunsTimeseries(ctx, p.ID,
		time.Now().Add(-time.Hour), time.Now().Add(time.Hour), "1 hour")
	require.NoError(t, err)
	require.NotNil(t, buckets)
	assert.Len(t, buckets, 0)
}
