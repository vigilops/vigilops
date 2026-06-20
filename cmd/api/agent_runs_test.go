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

func TestListRunsHandler_returns200WithEnvelopedArray(t *testing.T) {
	srv, projectID, key, app := newTestServer(t)
	ctx := context.Background()

	// Seed a run for the test project.
	require.NoError(t, app.store.AgentRuns.Insert(ctx, &store.AgentRun{
		ProjectID: uuid.MustParse(projectID),
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
	srvB, projectIDB, _, appB := newTestServer(t)
	ctx := context.Background()

	// A run that lives in tenant B
	runB := &store.AgentRun{ProjectID: uuid.MustParse(projectIDB), AgentName: "x", Status: "running"}
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

func TestRunHealthHandler_returns200WithRollup(t *testing.T) {
	srv, projectID, key, app := newTestServer(t)
	ctx := context.Background()
	pid := uuid.MustParse(projectID)

	run := &store.AgentRun{ProjectID: pid, AgentName: "health-agent", Status: "running"}
	require.NoError(t, app.store.AgentRuns.Insert(ctx, run))
	require.NoError(t, app.store.AgentRuns.Finish(ctx, run.ID, run.Timestamp, store.AgentRunFinish{
		Status:      "completed",
		TotalTokens: 1000,
	}))

	var body struct {
		Data []store.RunHealthRow `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodGet, srv.URL+"/v1/agent/health", key, nil, &body)

	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.NotEmpty(t, body.Data)

	var found *store.RunHealthRow
	for i := range body.Data {
		if body.Data[i].AgentName == "health-agent" {
			found = &body.Data[i]
		}
	}
	require.NotNil(t, found, "expected health-agent row")
	assert.Equal(t, 1, found.TotalRuns)
	assert.Equal(t, 1, found.CompletedRuns)
	assert.InDelta(t, 1.0, found.CompletionRate, 0.001)
}

func TestRunHealthHandler_returns401WithoutKey(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/agent/health", "", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRunsTimeseriesHandler_returns200Bucketed(t *testing.T) {
	srv, projectID, key, app := newTestServer(t)
	ctx := context.Background()
	pid := uuid.MustParse(projectID)

	run := &store.AgentRun{ProjectID: pid, AgentName: "ts", Status: "running"}
	require.NoError(t, app.store.AgentRuns.Insert(ctx, run))
	require.NoError(t, app.store.AgentRuns.Finish(ctx, run.ID, run.Timestamp, store.AgentRunFinish{
		Status: "completed",
	}))

	var body struct {
		Data []store.RunBucket `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodGet, srv.URL+"/v1/agent/runs/timeseries?bucket=1h", key, nil, &body)

	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", raw)
	require.NotEmpty(t, body.Data)
	assert.GreaterOrEqual(t, body.Data[0].Total, 1)
}

func TestRunsTimeseriesHandler_rejectsBadBucket(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/agent/runs/timeseries?bucket=99x", key, nil, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRunsTimeseriesHandler_returns401WithoutKey(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/v1/agent/runs/timeseries", "", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
