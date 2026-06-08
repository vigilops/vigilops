package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentStep struct {
	ID               uuid.UUID `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	ProjectID        uuid.UUID `json:"project_id"`
	AgentRunID       uuid.UUID `json:"agent_run_id"`
	StepIndex        int       `json:"step_index"`
	StepType         string    `json:"step_type"`
	Content          *string   `json:"content,omitempty"`
	ToolName         *string   `json:"tool_name,omitempty"`
	ToolInput        []byte    `json:"tool_input,omitempty"`
	ToolOutput       []byte    `json:"tool_output,omitempty"`
	ToolSuccess      *bool     `json:"tool_success,omitempty"`
	ToolLatencyMs    *int      `json:"tool_latency_ms,omitempty"`
	InputFingerprint []byte    `json:"input_fingerprint,omitempty"`
	Tokens           *int      `json:"tokens,omitempty"`
	CostUSD          *float64  `json:"cost_usd,omitempty"`
	Metadata         []byte    `json:"metadata,omitempty"`
}

type AgentStepStore struct {
	pool *pgxpool.Pool
}

func (s *AgentStepStore) Insert(ctx context.Context, st *AgentStep) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	if st.Timestamp.IsZero() {
		st.Timestamp = time.Now()
	}

	const q = `
		INSERT INTO agent_steps (
			timestamp, project_id, agent_run_id, step_index, step_type,
			content, tool_name, tool_input, tool_output, tool_success,
			tool_latency_ms, input_fingerprint, tokens, cost_usd, metadata
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15
		)
		RETURNING id
	`
	return s.pool.QueryRow(ctx, q,
		st.Timestamp, st.ProjectID, st.AgentRunID, st.StepIndex, st.StepType,
		st.Content, st.ToolName, st.ToolInput, st.ToolOutput, st.ToolSuccess,
		st.ToolLatencyMs, st.InputFingerprint, st.Tokens, st.CostUSD, st.Metadata,
	).Scan(&st.ID)
}

type LoopHit struct {
	Fingerprint []byte  `json:"fingerprint"`
	Hits        int     `json:"hits"`
	StepIndices []int   `json:"step_indices"`
	ToolName    *string `json:"tool_name,omitempty"`
}

func (s *AgentStepStore) ListLoops(ctx context.Context, projectID, runID uuid.UUID, from, to time.Time) ([]*LoopHit, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT input_fingerprint,
		       count(*)::int                              AS hits,
		       array_agg(step_index ORDER BY step_index) AS step_indices,
		       max(tool_name)                             AS tool_name
		FROM agent_steps
		WHERE project_id = $1 AND agent_run_id = $2
		  AND input_fingerprint IS NOT NULL
		  AND timestamp >= $3 AND timestamp < $4
		GROUP BY input_fingerprint
		HAVING count(*) >= 2
		ORDER BY hits DESC
	`
	rows, err := s.pool.Query(ctx, q, projectID, runID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*LoopHit
	for rows.Next() {
		h := &LoopHit{}
		if err := rows.Scan(&h.Fingerprint, &h.Hits, &h.StepIndices, &h.ToolName); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *AgentStepStore) ListByRun(ctx context.Context, projectID, runID uuid.UUID, from, to time.Time, limit int) ([]*AgentStep, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, timestamp, project_id, agent_run_id, step_index, step_type,
		       content, tool_name, tool_input, tool_output, tool_success,
		       tool_latency_ms, input_fingerprint, tokens, cost_usd, metadata
		FROM agent_steps
		WHERE project_id = $1 AND agent_run_id = $2
		  AND timestamp >= $3 AND timestamp < $4
		ORDER BY step_index ASC
		LIMIT $5
	`
	rows, err := s.pool.Query(ctx, q, projectID, runID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*AgentStep
	for rows.Next() {
		st := &AgentStep{}
		if err := rows.Scan(
			&st.ID, &st.Timestamp, &st.ProjectID, &st.AgentRunID, &st.StepIndex, &st.StepType,
			&st.Content, &st.ToolName, &st.ToolInput, &st.ToolOutput, &st.ToolSuccess,
			&st.ToolLatencyMs, &st.InputFingerprint, &st.Tokens, &st.CostUSD, &st.Metadata,
		); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// CountFingerprint returns how many times this fingerprint already appears in
// the run. Used by loop detection: ≥2 = repeated tool_name+input.
func (s *AgentStepStore) CountFingerprint(ctx context.Context, runID uuid.UUID, fingerprint []byte) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT count(*) FROM agent_steps
		WHERE agent_run_id = $1 AND input_fingerprint = $2
	`
	var n int
	if err := s.pool.QueryRow(ctx, q, runID, fingerprint).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
