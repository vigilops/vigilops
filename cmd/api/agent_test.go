package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/keelwave/keelwave/internal/store"
)

func TestListRunsHandler_returns200WithEnvelopedArray(t *testing.T) {
	srv, _, key, app := newTestServer(t)
	ctx := context.Background()

	// Seed a run for the test project.
	proj, _ := app.store.Projects.List(ctx)
	require.NotEmpty(t, proj)
	require.NoError(t, app.store.AgentRuns.Insert(ctx, &store.AgentRun{
		ProjectID: proj[0].ID,
		AgentName: "list-handler-test",
		Status:    "running",
	}))

	var body struct {
		Data []struct {
			ID        string    `json:"id"`
			AgentName string    `json:"agent_name"`
			Status    string    `json:"status"`
			Timestamp time.Time `json:"timestamp"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodGet, srv.URL+"/v1/agent/runs", key, nil, &body)

	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.NotEmpty(t, body.Data, "expected at least one run")
	assert.Equal(t, "list-handler-test", body.Data[0].AgentName)
}

func TestListRunsHandler_returns401WithoutKey(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/agent/runs", "", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetRunHandler_returns404OnCrossTenantID(t *testing.T) {
	srvA, _, keyA, appA := newTestServer(t)
	srvB, _, _, appB := newTestServer(t)
	ctx := context.Background()

	// A run that lives in tenant B
	projsB, _ := appB.store.Projects.List(ctx)
	require.NotEmpty(t, projsB)
	runB := &store.AgentRun{ProjectID: projsB[0].ID, AgentName: "x", Status: "running"}
	require.NoError(t, appB.store.AgentRuns.Insert(ctx, runB))

	// Tenant A queries it — must not see it.
	url := fmt.Sprintf("%s/v1/agent/runs/%s?at=%s", srvA.URL, runB.ID, runB.Timestamp.UTC().Format(time.RFC3339Nano))
	var body struct {
		Error string `json:"error"`
	}
	resp, raw := doJSON(t, http.MethodGet, url, keyA, nil, &body)

	require.Equal(t, http.StatusNotFound, resp.StatusCode, "body=%s", raw)
	assert.Equal(t, "not found", body.Error, "must be project-scoped not-found envelope, not chi 404")
	_ = srvB
	_ = appA
}

func TestListLoopsHandler_returnsHits3ForLoopingRun(t *testing.T) {
	srv, _, key, app := newTestServer(t)
	ctx := context.Background()
	projs, _ := app.store.Projects.List(ctx)
	require.NotEmpty(t, projs)
	pid := projs[0].ID

	run := &store.AgentRun{ProjectID: pid, AgentName: "looper", Status: "running"}
	require.NoError(t, app.store.AgentRuns.Insert(ctx, run))

	tool := "search"
	fp := []byte{0xaa, 0xbb, 0xcc}
	for i := 1; i <= 3; i++ {
		require.NoError(t, app.store.AgentSteps.Insert(ctx, &store.AgentStep{
			ProjectID:        pid,
			AgentRunID:       run.ID,
			StepIndex:        i,
			StepType:         "tool_call",
			ToolName:         &tool,
			InputFingerprint: fp,
		}))
	}

	url := fmt.Sprintf("%s/v1/agent/runs/%s/loops?at=%s",
		srv.URL, run.ID, run.Timestamp.UTC().Format(time.RFC3339Nano))
	var body struct {
		Data []struct {
			Hits        int    `json:"hits"`
			StepIndices []int  `json:"step_indices"`
			ToolName    string `json:"tool_name"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodGet, url, key, nil, &body)

	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.Len(t, body.Data, 1)
	assert.Equal(t, 3, body.Data[0].Hits)
	assert.Equal(t, []int{1, 2, 3}, body.Data[0].StepIndices)
	assert.Equal(t, "search", body.Data[0].ToolName)
}

func TestListStepsHandler_returnsStepsOrderedByIndex(t *testing.T) {
	srv, _, key, app := newTestServer(t)
	ctx := context.Background()
	projs, _ := app.store.Projects.List(ctx)
	require.NotEmpty(t, projs)
	pid := projs[0].ID

	run := &store.AgentRun{ProjectID: pid, AgentName: "stepper", Status: "running"}
	require.NoError(t, app.store.AgentRuns.Insert(ctx, run))
	for _, idx := range []int{3, 1, 2} {
		require.NoError(t, app.store.AgentSteps.Insert(ctx, &store.AgentStep{
			ProjectID:  pid,
			AgentRunID: run.ID,
			StepIndex:  idx,
			StepType:   "think",
		}))
	}

	url := fmt.Sprintf("%s/v1/agent/runs/%s/steps?at=%s",
		srv.URL, run.ID, run.Timestamp.UTC().Format(time.RFC3339Nano))
	var body struct {
		Data []struct {
			StepIndex int `json:"step_index"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodGet, url, key, nil, &body)

	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.Len(t, body.Data, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{body.Data[0].StepIndex, body.Data[1].StepIndex, body.Data[2].StepIndex})
}
