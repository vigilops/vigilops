package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/yusufnuru/vigil/internal/batch"
	"github.com/yusufnuru/vigil/internal/store"
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

// --- Append step --------------------------------------------------------

type ingestAgentStepPayload struct {
	Timestamp     *time.Time      `json:"timestamp,omitempty"`
	AgentRunID    uuid.UUID       `json:"agent_run_id"  validate:"required"`
	StepIndex     int             `json:"step_index"    validate:"gte=0"`
	StepType      string          `json:"step_type"     validate:"required,oneof=think tool_call tool_result replan"`
	Content       *string         `json:"content,omitempty"      validate:"omitempty,max=100000"`
	ToolName      *string         `json:"tool_name,omitempty"    validate:"omitempty,max=200"`
	ToolInput     json.RawMessage `json:"tool_input,omitempty"`
	ToolOutput    json.RawMessage `json:"tool_output,omitempty"`
	ToolSuccess   *bool           `json:"tool_success,omitempty"`
	ToolLatencyMs *int            `json:"tool_latency_ms,omitempty" validate:"omitempty,gte=0"`
	Tokens        *int            `json:"tokens,omitempty"          validate:"omitempty,gte=0"`
	CostUSD       *float64        `json:"cost_usd,omitempty"        validate:"omitempty,gte=0"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

// IngestAgentStep godoc
//
//	@Summary		Append an agent step
//	@Description	Records one step in an agent loop. Server computes a SHA-256 fingerprint of tool_name + tool_input for loop detection and bumps the agent_tools registry when tool_name is present.
//	@Tags			ingest
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		ingestAgentStepPayload	true	"Step payload"
//	@Success		201		{object}	ingestResponse
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/ingest/agent/steps [post]
func (app *application) ingestAgentStepHandler(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromContext(r.Context())

	var payload ingestAgentStepPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	step := &store.AgentStep{
		ID:               id,
		ProjectID:        projectID,
		AgentRunID:       payload.AgentRunID,
		StepIndex:        payload.StepIndex,
		StepType:         payload.StepType,
		Content:          payload.Content,
		ToolName:         payload.ToolName,
		ToolInput:        []byte(payload.ToolInput),
		ToolOutput:       []byte(payload.ToolOutput),
		ToolSuccess:      payload.ToolSuccess,
		ToolLatencyMs:    payload.ToolLatencyMs,
		Tokens:           payload.Tokens,
		CostUSD:          payload.CostUSD,
		Metadata:         []byte(payload.Metadata),
		InputFingerprint: stepFingerprint(payload.ToolName, payload.ToolInput),
	}
	if payload.Timestamp != nil {
		step.Timestamp = *payload.Timestamp
	} else {
		step.Timestamp = time.Now()
	}

	if err := app.batchers.AgentSteps.Enqueue(r.Context(), step); err != nil {
		if errors.Is(err, batch.ErrBufferFull) {
			app.serviceUnavailableResponse(w, r)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if payload.ToolName != nil {
		go app.upsertTool(projectID, *payload.ToolName)
	}

	if err := app.jsonResponse(w, http.StatusCreated, ingestResponse{ID: &step.ID, Timestamp: step.Timestamp}); err != nil {
		app.internalServerError(w, r, err)
	}
}

func stepFingerprint(toolName *string, toolInput json.RawMessage) []byte {
	if toolName == nil {
		return nil
	}
	h := sha256.New()
	h.Write([]byte(*toolName))
	h.Write([]byte{0})
	h.Write(toolInput)
	sum := h.Sum(nil)
	return sum
}

func (app *application) upsertTool(projectID uuid.UUID, toolName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.store.AgentTools.UpsertSeen(ctx, projectID, toolName); err != nil {
		app.logger.Warnw("agent_tools upsert failed", "err", err, "tool_name", toolName)
	}
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

// ListAgentSteps godoc
//
//	@Summary	List steps for an agent run, ordered by step_index asc
//	@Tags		agent
//	@Produce	json
//	@Param		runID	path		string	true	"Agent run UUID"
//	@Param		at		query		string	false	"RFC3339 hint for chunk pruning"
//	@Param		limit	query		int		false	"default 1000, max 1000"
//	@Success	200		{array}		store.AgentStep
//	@Failure	400		{object}	error
//	@Failure	401		{object}	error
//	@Failure	500		{object}	error
//	@Security	ApiKeyAuth
//	@Router		/agent/runs/{runID}/steps [get]
func (app *application) listAgentStepsHandler(w http.ResponseWriter, r *http.Request) {
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
	limit, err := parseIntInRange(r.URL.Query().Get("limit"), maxStepsLimit, 1, maxStepsLimit)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	steps, err := app.store.AgentSteps.ListByRun(r.Context(), projectID, runID, from, to, limit)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, steps); err != nil {
		app.internalServerError(w, r, err)
	}
}

// ListAgentLoops godoc
//
//	@Summary		Loop detection for an agent run
//	@Description	Returns each tool_name + input fingerprint repeated >= 2 times.
//	@Tags			agent
//	@Produce		json
//	@Param			runID	path		string	true	"Agent run UUID"
//	@Param			at		query		string	false	"RFC3339 hint for chunk pruning"
//	@Success		200		{array}		store.LoopHit
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/agent/runs/{runID}/loops [get]
func (app *application) listAgentLoopsHandler(w http.ResponseWriter, r *http.Request) {
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
	hits, err := app.store.AgentSteps.ListLoops(r.Context(), projectID, runID, from, to)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, hits); err != nil {
		app.internalServerError(w, r, err)
	}
}
