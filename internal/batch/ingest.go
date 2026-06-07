package batch

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/keelwave/keelwave/internal/store"
)

// Batchers owns one Buffer per hot-path hypertable.
//
// agent_runs is NOT batched. Its Finish handler does an UPDATE WHERE
// (id, timestamp) — those columns must already be in the DB. If the
// matching INSERT is still sitting in a buffer, Finish hits zero rows
// and returns 404.
type Batchers struct {
	AITraces     *Buffer[*store.AITrace]
	APIEvents    *Buffer[*store.APIEvent]
	InfraMetrics *Buffer[*store.InfraMetric]
	AgentSteps   *Buffer[*store.AgentStep]
}

func NewBatchers(pool *pgxpool.Pool, cfg Config, logger *zap.SugaredLogger) *Batchers {
	return &Batchers{
		AITraces:     newAITraces(pool, cfg, logger),
		APIEvents:    newAPIEvents(pool, cfg, logger),
		InfraMetrics: newInfraMetrics(pool, cfg, logger),
		AgentSteps:   newAgentSteps(pool, cfg, logger),
	}
}

func (b *Batchers) Start(ctx context.Context) {
	b.AITraces.Start(ctx)
	b.APIEvents.Start(ctx)
	b.InfraMetrics.Start(ctx)
	b.AgentSteps.Start(ctx)
}

// Stop drains every buffer under the shared ctx deadline.
func (b *Batchers) Stop(ctx context.Context) error {
	var firstErr error
	for _, buf := range []interface {
		Stop(ctx context.Context) error
	}{
		b.AITraces, b.APIEvents, b.InfraMetrics, b.AgentSteps,
	} {
		if err := buf.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

var aiTraceCols = []string{
	"id", "timestamp", "project_id", "model", "provider",
	"input_tokens", "output_tokens", "total_tokens", "cost_usd",
	"latency_ms", "status", "error_message", "request_id",
	"agent_run_id", "metadata",
}

func newAITraces(pool *pgxpool.Pool, cfg Config, logger *zap.SugaredLogger) *Buffer[*store.AITrace] {
	return New("ai_traces", cfg, logger, func(ctx context.Context, rows []*store.AITrace) error {
		src := pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
			r := rows[i]
			return []any{
				r.ID, r.Timestamp, r.ProjectID, r.Model, r.Provider,
				r.InputTokens, r.OutputTokens, r.TotalTokens, r.CostUSD,
				r.LatencyMs, r.Status, r.ErrorMessage, r.RequestID,
				r.AgentRunID, r.Metadata,
			}, nil
		})
		_, err := pool.CopyFrom(ctx, pgx.Identifier{"ai_traces"}, aiTraceCols, src)
		return err
	})
}

var apiEventCols = []string{
	"id", "timestamp", "project_id", "service", "method", "path",
	"status_code", "duration_ms", "request_size_bytes", "response_size_bytes",
	"ip", "user_agent", "error", "metadata",
}

func newAPIEvents(pool *pgxpool.Pool, cfg Config, logger *zap.SugaredLogger) *Buffer[*store.APIEvent] {
	return New("api_events", cfg, logger, func(ctx context.Context, rows []*store.APIEvent) error {
		src := pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
			r := rows[i]
			return []any{
				r.ID, r.Timestamp, r.ProjectID, r.Service, r.Method, r.Path,
				r.StatusCode, r.DurationMs, r.RequestSizeBytes, r.ResponseSizeBytes,
				r.IP, r.UserAgent, r.Error, r.Metadata,
			}, nil
		})
		_, err := pool.CopyFrom(ctx, pgx.Identifier{"api_events"}, apiEventCols, src)
		return err
	})
}

var infraMetricCols = []string{
	"timestamp", "project_id", "host", "metric_name", "value", "labels",
}

func newInfraMetrics(pool *pgxpool.Pool, cfg Config, logger *zap.SugaredLogger) *Buffer[*store.InfraMetric] {
	return New("infra_metrics", cfg, logger, func(ctx context.Context, rows []*store.InfraMetric) error {
		src := pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
			r := rows[i]
			return []any{
				r.Timestamp, r.ProjectID, r.Host, r.MetricName, r.Value, r.Labels,
			}, nil
		})
		_, err := pool.CopyFrom(ctx, pgx.Identifier{"infra_metrics"}, infraMetricCols, src)
		return err
	})
}

var agentStepCols = []string{
	"id", "timestamp", "project_id", "agent_run_id", "step_index", "step_type",
	"content", "tool_name", "tool_input", "tool_output", "tool_success",
	"tool_latency_ms", "input_fingerprint", "tokens", "cost_usd", "metadata",
}

func newAgentSteps(pool *pgxpool.Pool, cfg Config, logger *zap.SugaredLogger) *Buffer[*store.AgentStep] {
	return New("agent_steps", cfg, logger, func(ctx context.Context, rows []*store.AgentStep) error {
		src := pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
			r := rows[i]
			return []any{
				r.ID, r.Timestamp, r.ProjectID, r.AgentRunID, r.StepIndex, r.StepType,
				r.Content, r.ToolName, r.ToolInput, r.ToolOutput, r.ToolSuccess,
				r.ToolLatencyMs, r.InputFingerprint, r.Tokens, r.CostUSD, r.Metadata,
			}, nil
		})
		_, err := pool.CopyFrom(ctx, pgx.Identifier{"agent_steps"}, agentStepCols, src)
		return err
	})
}
