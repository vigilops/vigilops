package main

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/keelwave/keelwave/internal/batch"
	"github.com/keelwave/keelwave/internal/store"
)

type ingestEventPayload struct {
	Timestamp         *time.Time      `json:"timestamp,omitempty"`
	Service           string          `json:"service"             validate:"required,min=1,max=100"`
	Method            string          `json:"method"              validate:"required,oneof=GET POST PUT PATCH DELETE OPTIONS HEAD"`
	Path              string          `json:"path"                validate:"required,min=1,max=500"`
	StatusCode        int             `json:"status_code"         validate:"required,gte=100,lte=599"`
	DurationMs        int             `json:"duration_ms"         validate:"required,gte=0"`
	RequestSizeBytes  *int            `json:"request_size_bytes,omitempty"  validate:"omitempty,gte=0"`
	ResponseSizeBytes *int            `json:"response_size_bytes,omitempty" validate:"omitempty,gte=0"`
	IP                *string         `json:"ip,omitempty"        validate:"omitempty,ip"`
	UserAgent         *string         `json:"user_agent,omitempty" validate:"omitempty,max=500"`
	Error             *string         `json:"error,omitempty"     validate:"omitempty,max=2000"`
	Metadata          json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
}

// IngestEvent godoc
//
//	@Summary		Ingest an HTTP request event
//	@Description	Records one HTTP request: method, path, status, duration. project_id is derived from the API key.
//	@Tags			ingest
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		ingestEventPayload	true	"API event payload"
//	@Success		201		{object}	ingestResponse
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/ingest/events [post]
func (app *application) ingestEventHandler(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromContext(r.Context())

	var payload ingestEventPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var ip *net.IP
	if payload.IP != nil {
		parsed := net.ParseIP(*payload.IP)
		ip = &parsed
	}

	id, err := uuid.NewV7()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	e := &store.APIEvent{
		ID:                id,
		ProjectID:         projectID,
		Service:           payload.Service,
		Method:            payload.Method,
		Path:              payload.Path,
		StatusCode:        payload.StatusCode,
		DurationMs:        payload.DurationMs,
		RequestSizeBytes:  payload.RequestSizeBytes,
		ResponseSizeBytes: payload.ResponseSizeBytes,
		IP:                ip,
		UserAgent:         payload.UserAgent,
		Error:             payload.Error,
		Metadata:          []byte(payload.Metadata),
	}
	if payload.Timestamp != nil {
		e.Timestamp = *payload.Timestamp
	} else {
		e.Timestamp = time.Now()
	}

	if err := app.batchers.APIEvents.Enqueue(r.Context(), e); err != nil {
		if errors.Is(err, batch.ErrBufferFull) {
			app.serviceUnavailableResponse(w, r)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, ingestResponse{ID: &e.ID, Timestamp: e.Timestamp}); err != nil {
		app.internalServerError(w, r, err)
	}
}
