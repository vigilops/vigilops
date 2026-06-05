package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentEvaluation struct {
	ID           uuid.UUID `json:"id"`
	ProjectID    uuid.UUID `json:"project_id"`
	AgentRunID   uuid.UUID `json:"agent_run_id"`
	Correctness  *float64  `json:"correctness,omitempty"`
	Completeness *float64  `json:"completeness,omitempty"`
	Efficiency   *float64  `json:"efficiency,omitempty"`
	Safety       *float64  `json:"safety,omitempty"`
	Evaluator    string    `json:"evaluator"`
	EvaluatedAt  time.Time `json:"evaluated_at"`
	Notes        *string   `json:"notes,omitempty"`
}

type AgentEvaluationStore struct {
	pool *pgxpool.Pool
}

func (s *AgentEvaluationStore) Insert(ctx context.Context, e *AgentEvaluation) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		INSERT INTO agent_evaluations (
			project_id, agent_run_id,
			correctness, completeness, efficiency, safety,
			evaluator, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, evaluated_at
	`
	return s.pool.QueryRow(ctx, q,
		e.ProjectID, e.AgentRunID,
		e.Correctness, e.Completeness, e.Efficiency, e.Safety,
		e.Evaluator, e.Notes,
	).Scan(&e.ID, &e.EvaluatedAt)
}

func (s *AgentEvaluationStore) ListByRun(ctx context.Context, runID uuid.UUID) ([]*AgentEvaluation, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, project_id, agent_run_id,
		       correctness, completeness, efficiency, safety,
		       evaluator, evaluated_at, notes
		FROM agent_evaluations
		WHERE agent_run_id = $1
		ORDER BY evaluated_at DESC
	`
	rows, err := s.pool.Query(ctx, q, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*AgentEvaluation
	for rows.Next() {
		ev := &AgentEvaluation{}
		if err := rows.Scan(
			&ev.ID, &ev.ProjectID, &ev.AgentRunID,
			&ev.Correctness, &ev.Completeness, &ev.Efficiency, &ev.Safety,
			&ev.Evaluator, &ev.EvaluatedAt, &ev.Notes,
		); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}
