package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentEvaluationStore_Insert_assignsIDAndEvaluatedAt(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "eval")

	run := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, run))

	correctness := 0.95
	notes := "well-reasoned"
	ev := &AgentEvaluation{
		ProjectID:   p.ID,
		AgentRunID:  run.ID,
		Correctness: &correctness,
		Evaluator:   "claude-judge",
		Notes:       &notes,
	}
	require.NoError(t, s.AgentEvaluations.Insert(ctx, ev))
	assert.NotEqual(t, uuid.Nil, ev.ID)
	assert.False(t, ev.EvaluatedAt.IsZero())
}

func TestAgentEvaluationStore_ListByRun_returnsOnlyThatRunsEvals(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "evallist")

	r1 := &AgentRun{ProjectID: p.ID, AgentName: "a", Status: "running"}
	r2 := &AgentRun{ProjectID: p.ID, AgentName: "b", Status: "running"}
	require.NoError(t, s.AgentRuns.Insert(ctx, r1))
	require.NoError(t, s.AgentRuns.Insert(ctx, r2))

	for _, rid := range []uuid.UUID{r1.ID, r1.ID, r2.ID} {
		require.NoError(t, s.AgentEvaluations.Insert(ctx, &AgentEvaluation{
			ProjectID: p.ID, AgentRunID: rid, Evaluator: "j",
		}))
	}

	list, err := s.AgentEvaluations.ListByRun(ctx, r1.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}
