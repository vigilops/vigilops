package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/keelwave/keelwave/internal/store"
)

func TestListLoopsHandler_returnsHits3ForLoopingRun(t *testing.T) {
	ts := newTestServer(t)
	ctx := context.Background()
	pid := uuid.MustParse(ts.projID)

	run := &store.AgentRun{ProjectID: pid, AgentName: "looper", Status: "running"}
	require.NoError(t, ts.app.store.AgentRuns.Insert(ctx, run))

	tool := "search"
	fp := []byte{0xaa, 0xbb, 0xcc}
	for i := 1; i <= 3; i++ {
		require.NoError(t, ts.app.store.AgentSteps.Insert(ctx, &store.AgentStep{
			ProjectID:        pid,
			AgentRunID:       run.ID,
			StepIndex:        i,
			StepType:         "tool_call",
			ToolName:         &tool,
			InputFingerprint: fp,
		}))
	}

	url := fmt.Sprintf("%s/v1/projects/%s/agent/runs/%s/loops?at=%s",
		ts.srv.URL, ts.projID, run.ID, run.Timestamp.UTC().Format(time.RFC3339Nano))
	var body struct {
		Data []struct {
			Hits        int    `json:"hits"`
			StepIndices []int  `json:"step_indices"`
			ToolName    string `json:"tool_name"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodGet, url, ts.apiKey, nil, &body)

	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.Len(t, body.Data, 1)
	assert.Equal(t, 3, body.Data[0].Hits)
	assert.Equal(t, []int{1, 2, 3}, body.Data[0].StepIndices)
	assert.Equal(t, "search", body.Data[0].ToolName)
}

func TestListStepsHandler_returnsStepsOrderedByIndex(t *testing.T) {
	ts := newTestServer(t)
	ctx := context.Background()
	pid := uuid.MustParse(ts.projID)

	run := &store.AgentRun{ProjectID: pid, AgentName: "stepper", Status: "running"}
	require.NoError(t, ts.app.store.AgentRuns.Insert(ctx, run))
	for _, idx := range []int{3, 1, 2} {
		require.NoError(t, ts.app.store.AgentSteps.Insert(ctx, &store.AgentStep{
			ProjectID:  pid,
			AgentRunID: run.ID,
			StepIndex:  idx,
			StepType:   "think",
		}))
	}

	url := fmt.Sprintf("%s/v1/projects/%s/agent/runs/%s/steps?at=%s",
		ts.srv.URL, ts.projID, run.ID, run.Timestamp.UTC().Format(time.RFC3339Nano))
	var body struct {
		Data []struct {
			StepIndex int `json:"step_index"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodGet, url, ts.apiKey, nil, &body)

	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.Len(t, body.Data, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{body.Data[0].StepIndex, body.Data[1].StepIndex, body.Data[2].StepIndex})
}

func TestToolStatsHandler_returns200WithAggregates(t *testing.T) {
	ts := newTestServer(t)
	ctx := context.Background()
	pid := uuid.MustParse(ts.projID)

	run := &store.AgentRun{ProjectID: pid, AgentName: "tool-stats", Status: "running"}
	require.NoError(t, ts.app.store.AgentRuns.Insert(ctx, run))

	search := "search"
	steps := []*store.AgentStep{
		{ToolName: &search, ToolSuccess: new(true), ToolLatencyMs: new(10)},
		{ToolName: &search, ToolSuccess: new(false), ToolLatencyMs: new(20)},
	}
	for i, st := range steps {
		st.ProjectID = pid
		st.AgentRunID = run.ID
		st.StepIndex = i + 1
		st.StepType = "tool_call"
		require.NoError(t, ts.app.store.AgentSteps.Insert(ctx, st))
	}

	var body struct {
		Data []store.ToolStat `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodGet, ts.srv.URL+"/v1/projects/"+ts.projID+"/agent/tools/stats", ts.apiKey, nil, &body)

	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.Len(t, body.Data, 1)
	assert.Equal(t, "search", body.Data[0].ToolName)
	assert.Equal(t, 2, body.Data[0].CallCount)
	assert.Equal(t, 1, body.Data[0].SuccessCount)
	assert.Equal(t, 1, body.Data[0].FailCount)
}

func TestToolStatsHandler_returns401WithoutKey(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, ts.srv.URL+"/v1/projects/"+ts.projID+"/agent/tools/stats", "", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestStepDistributionHandler_returns200(t *testing.T) {
	ts := newTestServer(t)
	ctx := context.Background()
	pid := uuid.MustParse(ts.projID)
	run := &store.AgentRun{ProjectID: pid, AgentName: "a", Status: "running"}
	require.NoError(t, ts.app.store.AgentRuns.Insert(ctx, run))
	require.NoError(t, ts.app.store.AgentSteps.Insert(ctx, &store.AgentStep{
		ProjectID: pid, AgentRunID: run.ID, StepIndex: 1, StepType: "think",
	}))
	var body struct{ Data []store.StepTypeCount `json:"data"` }
	resp, raw := doJSON(t, http.MethodGet, ts.srv.URL+"/v1/projects/"+ts.projID+"/agent/steps/distribution", ts.apiKey, nil, &body)
	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.NotEmpty(t, body.Data)
}
