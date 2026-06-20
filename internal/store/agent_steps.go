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

	out := make([]*LoopHit, 0)
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

	out := make([]*AgentStep, 0)
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

type ToolStat struct {
	ToolName     string  `json:"tool_name"`
	CallCount    int     `json:"call_count"`
	SuccessCount int     `json:"success_count"`
	FailCount    int     `json:"fail_count"`
	SuccessRate  float64 `json:"success_rate"`
	P95LatencyMs int     `json:"p95_latency_ms"`
}

// ToolStats aggregates tool usage across every run in the project, scoped to a
// time window. tool_success may be null; null counts toward neither success nor
// failure but still counts as a call. p95 ignores null latencies.
func (s *AgentStepStore) ToolStats(ctx context.Context, projectID uuid.UUID, from, to time.Time) ([]*ToolStat, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT tool_name,
		       count(*)::int                                              AS call_count,
		       count(*) FILTER (WHERE tool_success IS TRUE)::int          AS success_count,
		       count(*) FILTER (WHERE tool_success IS FALSE)::int         AS fail_count,
		       coalesce(
		           percentile_disc(0.95) WITHIN GROUP (ORDER BY tool_latency_ms), 0
		       )::int                                                     AS p95_latency_ms
		FROM agent_steps
		WHERE project_id = $1
		  AND tool_name IS NOT NULL
		  AND timestamp >= $2 AND timestamp < $3
		GROUP BY tool_name
		ORDER BY call_count DESC
	`
	rows, err := s.pool.Query(ctx, q, projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*ToolStat, 0)
	for rows.Next() {
		ts := &ToolStat{}
		if err := rows.Scan(&ts.ToolName, &ts.CallCount, &ts.SuccessCount, &ts.FailCount, &ts.P95LatencyMs); err != nil {
			return nil, err
		}
		if ts.CallCount > 0 {
			ts.SuccessRate = float64(ts.SuccessCount) / float64(ts.CallCount)
		}
		out = append(out, ts)
	}
	return out, rows.Err()
}
