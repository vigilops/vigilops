package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestAI_requiresAuth(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/ai", "",
		map[string]any{"model": "m", "status": "success"}, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestIngestAI_validatesStatusEnum(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/ai", key,
		map[string]any{"model": "m", "status": "weird"}, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIngestAI_validInsertReturns201WithID(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/ai", key,
		map[string]any{"model": "m", "status": "success"}, &body)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "body=%s", raw)
	assert.NotEmpty(t, body.Data.ID)
}

func TestIngestEvents_validInsertReturns201(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/events", key,
		map[string]any{
			"service": "api", "method": "GET", "path": "/x",
			"status_code": 200, "duration_ms": 12,
		}, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestIngestEvents_rejectsInvalidMethod(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/events", key,
		map[string]any{
			"service": "api", "method": "WRONG", "path": "/x",
			"status_code": 200, "duration_ms": 1,
		}, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIngestMetrics_validInsertReturns201(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/metrics", key,
		map[string]any{"host": "h1", "metric_name": "cpu", "value": 42.5}, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestIngestAgentRunStart_returns201WithIDAndTimestamp(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	var body struct {
		Data struct {
			ID        string `json:"id"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
	}
	resp, raw := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/agent/runs", key,
		map[string]any{"agent_name": "a", "input": "x"}, &body)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "body=%s", raw)
	assert.NotEmpty(t, body.Data.ID)
	assert.NotEmpty(t, body.Data.Timestamp)
}

func TestIngestAgentStep_validReturns201(t *testing.T) {
	srv, _, key, _ := newTestServer(t)

	// Start a run first.
	var run struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/agent/runs", key,
		map[string]any{"agent_name": "a"}, &run)

	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/agent/steps", key,
		map[string]any{
			"agent_run_id": run.Data.ID,
			"step_index":   1,
			"step_type":    "think",
		}, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestIngestAgentStep_rejectsBadStepType(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/agent/steps", key,
		map[string]any{
			"agent_run_id": "00000000-0000-0000-0000-000000000000",
			"step_index":   1,
			"step_type":    "lalala",
		}, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIngestAgentRunFinish_204(t *testing.T) {
	srv, _, key, _ := newTestServer(t)

	var run struct {
		Data struct {
			ID        string `json:"id"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
	}
	doJSON(t, http.MethodPost, srv.URL+"/v1/ingest/agent/runs", key,
		map[string]any{"agent_name": "a"}, &run)

	resp, _ := doJSON(t, http.MethodPost,
		srv.URL+"/v1/ingest/agent/runs/"+run.Data.ID+"/finish", key,
		map[string]any{
			"timestamp":          run.Data.Timestamp,
			"status":             "completed",
			"termination_reason": "clean",
			"total_steps":        1,
			"total_tokens":       10,
		}, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestIngestAgentRunFinish_404OnUnknownRun(t *testing.T) {
	srv, _, key, _ := newTestServer(t)
	resp, _ := doJSON(t, http.MethodPost,
		srv.URL+"/v1/ingest/agent/runs/00000000-0000-0000-0000-000000000000/finish", key,
		map[string]any{
			"timestamp":    "2026-06-08T00:00:00Z",
			"status":       "completed",
			"total_steps":  0,
			"total_tokens": 0,
		}, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
