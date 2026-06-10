package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (s *AgentRunStore) ListByProject(ctx context.Context, projectID uuid.UUID, from, to time.Time, limit, offset int) ([]*AgentRun, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, timestamp, project_id, agent_name, status,
		       termination_reason, loop_detected, loop_step_index,
		       total_steps, total_tokens, total_cost_usd, duration_ms,
		       input, output, metadata, finished_at
		FROM agent_runs
		WHERE project_id = $1 AND timestamp >= $2 AND timestamp < $3
		ORDER BY timestamp DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := s.pool.Query(ctx, q, projectID, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*AgentRun, 0)
	for rows.Next() {
		r := &AgentRun{}
		if err := rows.Scan(
			&r.ID, &r.Timestamp, &r.ProjectID, &r.AgentName, &r.Status,
			&r.TerminationReason, &r.LoopDetected, &r.LoopStepIndex,
			&r.TotalSteps, &r.TotalTokens, &r.TotalCostUSD, &r.DurationMs,
			&r.Input, &r.Output, &r.Metadata, &r.FinishedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *AgentRunStore) GetByID(ctx context.Context, projectID, runID uuid.UUID, from, to time.Time) (*AgentRun, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, timestamp, project_id, agent_name, status,
		       termination_reason, loop_detected, loop_step_index,
		       total_steps, total_tokens, total_cost_usd, duration_ms,
		       input, output, metadata, finished_at
		FROM agent_runs
		WHERE project_id = $1 AND id = $2
		  AND timestamp >= $3 AND timestamp < $4
		LIMIT 1
	`
	r := &AgentRun{}
	err := s.pool.QueryRow(ctx, q, projectID, runID, from, to).Scan(
		&r.ID, &r.Timestamp, &r.ProjectID, &r.AgentName, &r.Status,
		&r.TerminationReason, &r.LoopDetected, &r.LoopStepIndex,
		&r.TotalSteps, &r.TotalTokens, &r.TotalCostUSD, &r.DurationMs,
		&r.Input, &r.Output, &r.Metadata, &r.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return r, nil
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
