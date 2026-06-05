package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AITrace struct {
	ID           uuid.UUID  `json:"id"`
	Timestamp    time.Time  `json:"timestamp"`
	ProjectID    uuid.UUID  `json:"project_id"`
	Model        string     `json:"model"`
	Provider     *string    `json:"provider,omitempty"`
	InputTokens  *int       `json:"input_tokens,omitempty"`
	OutputTokens *int       `json:"output_tokens,omitempty"`
	TotalTokens  *int       `json:"total_tokens,omitempty"`
	CostUSD      *float64   `json:"cost_usd,omitempty"`
	LatencyMs    *int       `json:"latency_ms,omitempty"`
	Status       string     `json:"status"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	RequestID    *string    `json:"request_id,omitempty"`
	AgentRunID   *uuid.UUID `json:"agent_run_id,omitempty"`
	Metadata     []byte     `json:"metadata,omitempty"`
}

type AITraceStore struct {
	pool *pgxpool.Pool
}

func (s *AITraceStore) Insert(ctx context.Context, t *AITrace) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	if t.Timestamp.IsZero() {
		t.Timestamp = time.Now()
	}

	const q = `
		INSERT INTO ai_traces (
			timestamp, project_id, model, provider,
			input_tokens, output_tokens, total_tokens, cost_usd,
			latency_ms, status, error_message, request_id,
			agent_run_id, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14
		)
		RETURNING id
	`
	return s.pool.QueryRow(ctx, q,
		t.Timestamp, t.ProjectID, t.Model, t.Provider,
		t.InputTokens, t.OutputTokens, t.TotalTokens, t.CostUSD,
		t.LatencyMs, t.Status, t.ErrorMessage, t.RequestID,
		t.AgentRunID, t.Metadata,
	).Scan(&t.ID)
}
