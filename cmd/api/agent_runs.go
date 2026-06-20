package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/keelwave/keelwave/internal/store"
)

type ingestAgentRunStartPayload struct {
	Timestamp *time.Time      `json:"timestamp,omitempty"`
	AgentName string          `json:"agent_name"      validate:"required,min=1,max=200"`
	Input     *string         `json:"input,omitempty" validate:"omitempty,max=10000"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// IngestAgentRunStart godoc
//
//	@Summary		Start an agent run
//	@Description	Opens a new agent run record with status="running". Returns id + timestamp; the client must pass both back to finish the run.
//	@Tags			ingest
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		ingestAgentRunStartPayload	true	"Run start payload"
//	@Success		201		{object}	ingestResponse
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/ingest/agent/runs [post]
func (app *application) ingestAgentRunStartHandler(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromContext(r.Context())

	var payload ingestAgentRunStartPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	run := &store.AgentRun{
		ProjectID: projectID,
		AgentName: payload.AgentName,
		Status:    "running",
		Input:     payload.Input,
		Metadata:  []byte(payload.Metadata),
	}
	if payload.Timestamp != nil {
		run.Timestamp = *payload.Timestamp
	}

	if err := app.store.AgentRuns.Insert(r.Context(), run); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, ingestResponse{ID: &run.ID, Timestamp: run.Timestamp}); err != nil {
		app.internalServerError(w, r, err)
	}
}

// --- Finish run ---------------------------------------------------------

type ingestAgentRunFinishPayload struct {
	Timestamp         time.Time `json:"timestamp"             validate:"required"`
	Status            string    `json:"status"                validate:"required,oneof=completed failed"`
	TerminationReason *string   `json:"termination_reason,omitempty" validate:"omitempty,oneof=clean max_steps_reached context_limit error loop_detected timeout"`
	LoopDetected      bool      `json:"loop_detected"`
	LoopStepIndex     *int      `json:"loop_step_index,omitempty" validate:"omitempty,gte=0"`
	TotalSteps        int       `json:"total_steps"           validate:"gte=0"`
	TotalTokens       int       `json:"total_tokens"          validate:"gte=0"`
	TotalCostUSD      *float64  `json:"total_cost_usd,omitempty" validate:"omitempty,gte=0"`
	DurationMs        *int      `json:"duration_ms,omitempty" validate:"omitempty,gte=0"`
	Output            *string   `json:"output,omitempty"      validate:"omitempty,max=100000"`
}

// IngestAgentRunFinish godoc
//
//	@Summary		Finish an agent run
//	@Description	Marks a run terminal and writes totals. timestamp must match the value returned at start (hypertable PK).
//	@Tags			ingest
//	@Accept			json
//	@Produce		json
//	@Param			runID	path	string						true	"Agent run UUID"
//	@Param			payload	body	ingestAgentRunFinishPayload	true	"Run finish payload"
//	@Success		204
//	@Failure		400	{object}	error
//	@Failure		401	{object}	error
//	@Failure		404	{object}	error
//	@Failure		500	{object}	error
//	@Security		ApiKeyAuth
//	@Router			/ingest/agent/runs/{runID}/finish [post]
func (app *application) ingestAgentRunFinishHandler(w http.ResponseWriter, r *http.Request) {
	runID, err := parseUUIDParam(r, "runID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var payload ingestAgentRunFinishPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.store.AgentRuns.Finish(r.Context(), runID, payload.Timestamp, store.AgentRunFinish{
		Status:            payload.Status,
		TerminationReason: payload.TerminationReason,
		LoopDetected:      payload.LoopDetected,
		LoopStepIndex:     payload.LoopStepIndex,
		TotalSteps:        payload.TotalSteps,
		TotalTokens:       payload.TotalTokens,
		TotalCostUSD:      payload.TotalCostUSD,
		DurationMs:        payload.DurationMs,
		Output:            payload.Output,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.noContentResponse(w)
}

// --- Reads --------------------------------------------------------------

// ListAgentRuns godoc
//
//	@Summary	List agent runs for the calling project
//	@Tags		agent
//	@Produce	json
//	@Param		from	query		string	false	"RFC3339, default now - 30d"
//	@Param		to		query		string	false	"RFC3339, default now"
//	@Param		limit	query		int		false	"default 25, max 1000"
//	@Param		offset	query		int		false	"default 0"
//	@Success	200		{array}		store.AgentRun
//	@Failure	400		{object}	error
//	@Failure	401		{object}	error
//	@Failure	500		{object}	error
//	@Security	ApiKeyAuth
//	@Router		/agent/runs [get]
func (app *application) listAgentRunsHandler(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromContext(r.Context())
	params, err := parseListParams(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	runs, err := app.store.AgentRuns.ListByProject(r.Context(), projectID, params.From, params.To, params.Limit, params.Offset)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, runs); err != nil {
		app.internalServerError(w, r, err)
	}
}

// GetAgentRun godoc
//
//	@Summary	Fetch a single agent run by id
//	@Tags		agent
//	@Produce	json
//	@Param		runID	path		string	true	"Agent run UUID"
//	@Param		at		query		string	false	"RFC3339; if set, lookup is bounded to a 2s window for fast chunk pruning"
//	@Success	200		{object}	store.AgentRun
//	@Failure	400		{object}	error
//	@Failure	401		{object}	error
//	@Failure	404		{object}	error
//	@Failure	500		{object}	error
//	@Security	ApiKeyAuth
//	@Router		/agent/runs/{runID} [get]
func (app *application) getAgentRunHandler(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromContext(r.Context())
	runID, err := parseUUIDParam(r, "runID")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	from, to, err := parseAtWindow(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	run, err := app.store.AgentRuns.GetByID(r.Context(), projectID, runID, from, to)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, run); err != nil {
		app.internalServerError(w, r, err)
	}
}

// RunHealth godoc
//
//	@Summary		Per-agent run-health rollup for the calling project
//	@Description	Rolls up run outcomes per agent_name: total runs, completion rate, loop rate, avg cost, and avg tokens over the window.
//	@Tags			agent
//	@Produce		json
//	@Param			from	query		string	false	"RFC3339, default now - 30d"
//	@Param			to		query		string	false	"RFC3339, default now"
//	@Success		200		{array}		store.RunHealthRow
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/agent/health [get]
func (app *application) runHealthHandler(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromContext(r.Context())
	params, err := parseListParams(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	rows, err := app.store.AgentRuns.RunHealth(r.Context(), projectID, params.From, params.To)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, rows); err != nil {
		app.internalServerError(w, r, err)
	}
}

var bucketIntervals = map[string]string{
	"1h": "1 hour",
	"6h": "6 hours",
	"1d": "1 day",
}

// RunsTimeseries godoc
//
//	@Summary		Bucketed run counts over time for the calling project
//	@Description	Returns per-bucket run outcome counts (total, completed, failed, loop) via TimescaleDB time_bucket. bucket is one of 1h, 6h, 1d.
//	@Tags			agent
//	@Produce		json
//	@Param			from	query		string	false	"RFC3339, default now - 30d"
//	@Param			to		query		string	false	"RFC3339, default now"
//	@Param			bucket	query		string	false	"bucket size: 1h (default), 6h, 1d"
//	@Success		200		{array}		store.RunBucket
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/agent/runs/timeseries [get]
func (app *application) runsTimeseriesHandler(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromContext(r.Context())
	params, err := parseListParams(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		bucket = "1h"
	}
	interval, ok := bucketIntervals[bucket]
	if !ok {
		app.badRequestResponse(w, r, errors.New("bucket must be one of: 1h, 6h, 1d"))
		return
	}

	buckets, err := app.store.AgentRuns.RunsTimeseries(r.Context(), projectID, params.From, params.To, interval)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, buckets); err != nil {
		app.internalServerError(w, r, err)
	}
}
