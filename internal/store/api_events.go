package store

import (
	"context"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIEvent struct {
	ID                uuid.UUID `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	ProjectID         uuid.UUID `json:"project_id"`
	Service           string    `json:"service"`
	Method            string    `json:"method"`
	Path              string    `json:"path"`
	StatusCode        int       `json:"status_code"`
	DurationMs        int       `json:"duration_ms"`
	RequestSizeBytes  *int      `json:"request_size_bytes,omitempty"`
	ResponseSizeBytes *int      `json:"response_size_bytes,omitempty"`
	IP                *net.IP   `json:"ip,omitempty"`
	UserAgent         *string   `json:"user_agent,omitempty"`
	Error             *string   `json:"error,omitempty"`
	Metadata          []byte    `json:"metadata,omitempty"`
}

type APIEventStore struct {
	pool *pgxpool.Pool
}

func (s *APIEventStore) Insert(ctx context.Context, e *APIEvent) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	const q = `
		INSERT INTO api_events (
			timestamp, project_id, service, method, path,
			status_code, duration_ms, request_size_bytes, response_size_bytes,
			ip, user_agent, error, metadata
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13
		)
		RETURNING id
	`
	return s.pool.QueryRow(ctx, q,
		e.Timestamp, e.ProjectID, e.Service, e.Method, e.Path,
		e.StatusCode, e.DurationMs, e.RequestSizeBytes, e.ResponseSizeBytes,
		e.IP, e.UserAgent, e.Error, e.Metadata,
	).Scan(&e.ID)
}
