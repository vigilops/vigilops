package main

import (
	"time"

	"github.com/google/uuid"
)

// ingestResponse is the 201 envelope returned by every ingest endpoint that
// produces a single row. ID is a pointer + omitempty so InfraMetrics (composite
// PK, no id) can reuse the type and emit only the timestamp.
type ingestResponse struct {
	ID        *uuid.UUID `json:"id,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}
