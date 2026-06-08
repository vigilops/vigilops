package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/keelwave/keelwave/internal/batch"
	"github.com/keelwave/keelwave/internal/store"
)

type ingestMetricPayload struct {
	Timestamp  *time.Time      `json:"timestamp,omitempty"`
	Host       string          `json:"host"        validate:"required,min=1,max=200"`
	MetricName string          `json:"metric_name" validate:"required,min=1,max=200"`
	Value      float64         `json:"value"       validate:"required"`
	Labels     json.RawMessage `json:"labels,omitempty"`
}

// IngestMetric godoc
//
//	@Summary		Ingest a host metric
//	@Description	Records one infrastructure metric sample (cpu_percent, memory_used, etc.). project_id is derived from the API key.
//	@Tags			ingest
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		ingestMetricPayload	true	"Metric payload"
//	@Success		201		{object}	ingestResponse
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/ingest/metrics [post]
func (app *application) ingestMetricHandler(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromContext(r.Context())

	var payload ingestMetricPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	m := &store.InfraMetric{
		ProjectID:  projectID,
		Host:       payload.Host,
		MetricName: payload.MetricName,
		Value:      payload.Value,
		Labels:     []byte(payload.Labels),
	}
	if payload.Timestamp != nil {
		m.Timestamp = *payload.Timestamp
	} else {
		m.Timestamp = time.Now()
	}

	if err := app.batchers.InfraMetrics.Enqueue(r.Context(), m); err != nil {
		if errors.Is(err, batch.ErrBufferFull) {
			app.serviceUnavailableResponse(w, r)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, ingestResponse{Timestamp: m.Timestamp}); err != nil {
		app.internalServerError(w, r, err)
	}
}
