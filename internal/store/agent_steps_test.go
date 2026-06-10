package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentStepStore_ListByRun_ordersByStepIndexAsc(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "steplist")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	for _, idx := range []int{3, 1, 2} {
		step := &AgentStep{
			ProjectID:  p.ID,
			AgentRunID: run.ID,
			StepIndex:  idx,
			StepType:   "think",
		}
		require.NoError(t, s.AgentSteps.Insert(ctx, step))
	}

	from := run.Timestamp.Add(-time.Hour)
	to := run.Timestamp.Add(time.Hour)
	steps, err := s.AgentSteps.ListByRun(ctx, p.ID, run.ID, from, to, 100)
	require.NoError(t, err)
	require.Len(t, steps, 3)

	assert.Equal(t, 1, steps[0].StepIndex)
	assert.Equal(t, 2, steps[1].StepIndex)
	assert.Equal(t, 3, steps[2].StepIndex)
}

func TestAgentStepStore_Insert_assignsIDAndDefaultTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "stepinsert")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	st := &AgentStep{
		ProjectID:  p.ID,
		AgentRunID: run.ID,
		StepIndex:  1,
		StepType:   "think",
	}
	require.NoError(t, s.AgentSteps.Insert(ctx, st))
	assert.NotEqual(t, uuid.UUID{}, st.ID)
	assert.False(t, st.Timestamp.IsZero())
}

func TestAgentStepStore_CountFingerprint_countsExactMatches(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "stepcount")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	fp := []byte{0x01, 0x02, 0x03}
	for i := 0; i < 3; i++ {
		require.NoError(t, s.AgentSteps.Insert(ctx, &AgentStep{
			ProjectID: p.ID, AgentRunID: run.ID,
			StepIndex: i + 1, StepType: "tool_call",
			InputFingerprint: fp,
		}))
	}

	n, err := s.AgentSteps.CountFingerprint(ctx, run.ID, fp)
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	m, err := s.AgentSteps.CountFingerprint(ctx, run.ID, []byte{0xff})
	require.NoError(t, err)
	assert.Equal(t, 0, m)
}

func TestAgentStepStore_ListLoops_groupsByFingerprint_filtersBelowTwo(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "loops")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	toolName := "search"
	repeat := []byte{0x01, 0x02, 0x03}
	once := []byte{0xff, 0xee, 0xdd}

	for i, fp := range [][]byte{repeat, repeat, repeat, once} {
		step := &AgentStep{
			ProjectID:        p.ID,
			AgentRunID:       run.ID,
			StepIndex:        i + 1,
			StepType:         "tool_call",
			ToolName:         &toolName,
			InputFingerprint: fp,
		}
		require.NoError(t, s.AgentSteps.Insert(ctx, step))
	}

	from := run.Timestamp.Add(-time.Hour)
	to := run.Timestamp.Add(time.Hour)
	hits, err := s.AgentSteps.ListLoops(ctx, p.ID, run.ID, from, to)
	require.NoError(t, err)
	require.Len(t, hits, 1, "only the repeated fingerprint qualifies as a loop")

	assert.Equal(t, repeat, hits[0].Fingerprint)
	assert.Equal(t, 3, hits[0].Hits)
	assert.Equal(t, []int{1, 2, 3}, hits[0].StepIndices)
	require.NotNil(t, hits[0].ToolName)
	assert.Equal(t, "search", *hits[0].ToolName)
}

func TestAgentStepStore_ListLoops_returnsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "loops-empty")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	hits, err := s.AgentSteps.ListLoops(ctx, p.ID, run.ID,
		run.Timestamp.Add(-time.Hour), run.Timestamp.Add(time.Hour))
	require.NoError(t, err)
	require.NotNil(t, hits, "must return [] not nil so JSON renders as array")
	assert.Len(t, hits, 0)
}

func TestAgentStepStore_ListByRun_returnsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "steps-empty")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	steps, err := s.AgentSteps.ListByRun(ctx, p.ID, run.ID,
		run.Timestamp.Add(-time.Hour), run.Timestamp.Add(time.Hour), 100)
	require.NoError(t, err)
	require.NotNil(t, steps)
	assert.Len(t, steps, 0)
}
