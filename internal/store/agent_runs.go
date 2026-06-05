package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentRun struct {
	ID                uuid.UUID  `json:"id"`
	Timestamp         time.Time  `json:"timestamp"`
	ProjectID         uuid.UUID  `json:"project_id"`
	AgentName         string     `json:"agent_name"`
	Status            string     `json:"status"`
	TerminationReason *string    `json:"termination_reason,omitempty"`
	LoopDetected      bool       `json:"loop_detected"`
	LoopStepIndex     *int       `json:"loop_step_index,omitempty"`
	TotalSteps        int        `json:"total_steps"`
	TotalTokens       int        `json:"total_tokens"`
	TotalCostUSD      *float64   `json:"total_cost_usd,omitempty"`
	DurationMs        *int       `json:"duration_ms,omitempty"`
	Input             *string    `json:"input,omitempty"`
	Output            *string    `json:"output,omitempty"`
	Metadata          []byte     `json:"metadata,omitempty"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}

type AgentRunFinish struct {
	Status            string
	TerminationReason *string
	LoopDetected      bool
	LoopStepIndex     *int
	TotalSteps        int
	TotalTokens       int
	TotalCostUSD      *float64
	DurationMs        *int
	Output            *string
}

type AgentRunStore struct {
	pool *pgxpool.Pool
}

func (s *AgentRunStore) Insert(ctx context.Context, r *AgentRun) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}

	const q = `
		INSERT INTO agent_runs (
			timestamp, project_id, agent_name, status,
			input, metadata
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return s.pool.QueryRow(ctx, q,
		r.Timestamp, r.ProjectID, r.AgentName, r.Status,
		r.Input, r.Metadata,
	).Scan(&r.ID)
}

// Finish marks a run terminal. Caller passes id + timestamp because hypertable
// PK is (id, timestamp); without timestamp the UPDATE has to scan every chunk.
func (s *AgentRunStore) Finish(ctx context.Context, id uuid.UUID, ts time.Time, f AgentRunFinish) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		UPDATE agent_runs SET
			status              = $3,
			termination_reason  = $4,
			loop_detected       = $5,
			loop_step_index     = $6,
			total_steps         = $7,
			total_tokens        = $8,
			total_cost_usd      = $9,
			duration_ms         = $10,
			output              = $11,
			finished_at         = now()
		WHERE id = $1 AND timestamp = $2
	`
	tag, err := s.pool.Exec(ctx, q,
		id, ts, f.Status, f.TerminationReason, f.LoopDetected, f.LoopStepIndex,
		f.TotalSteps, f.TotalTokens, f.TotalCostUSD, f.DurationMs, f.Output,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
