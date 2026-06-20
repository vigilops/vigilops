package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource already exists")
	QueryTimeoutDuration = 5 * time.Second
)

type Storage struct {
	Projects interface {
		Create(ctx context.Context, p *Project) error
		GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
		List(ctx context.Context) ([]*Project, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}
	APIKeys interface {
		Create(ctx context.Context, k *APIKey) error
		GetByHash(ctx context.Context, hash []byte) (*APIKey, error)
		TouchLastUsed(ctx context.Context, id uuid.UUID) error
		ListByProject(ctx context.Context, projectID uuid.UUID) ([]*APIKey, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}
	AITraces interface {
		Insert(ctx context.Context, t *AITrace) error
	}
	APIEvents interface {
		Insert(ctx context.Context, e *APIEvent) error
	}
	InfraMetrics interface {
		Insert(ctx context.Context, m *InfraMetric) error
	}
	AgentRuns interface {
		Insert(ctx context.Context, r *AgentRun) error
		Finish(ctx context.Context, id uuid.UUID, ts time.Time, f AgentRunFinish) error
		GetByID(ctx context.Context, projectID, runID uuid.UUID, from, to time.Time) (*AgentRun, error)
		ListByProject(ctx context.Context, projectID uuid.UUID, from, to time.Time, limit, offset int) ([]*AgentRun, error)
		RunHealth(ctx context.Context, projectID uuid.UUID, from, to time.Time) ([]*RunHealthRow, error)
		RunsTimeseries(ctx context.Context, projectID uuid.UUID, from, to time.Time, interval string) ([]*RunBucket, error)
	}
	AgentSteps interface {
		Insert(ctx context.Context, st *AgentStep) error
		CountFingerprint(ctx context.Context, runID uuid.UUID, fingerprint []byte) (int, error)
		ListByRun(ctx context.Context, projectID, runID uuid.UUID, from, to time.Time, limit int) ([]*AgentStep, error)
		ListLoops(ctx context.Context, projectID, runID uuid.UUID, from, to time.Time) ([]*LoopHit, error)
		ToolStats(ctx context.Context, projectID uuid.UUID, from, to time.Time) ([]*ToolStat, error)
	}
	AgentTools interface {
		UpsertSeen(ctx context.Context, projectID uuid.UUID, toolName string) error
		ListByProject(ctx context.Context, projectID uuid.UUID) ([]*AgentTool, error)
	}
	AgentEvaluations interface {
		Insert(ctx context.Context, e *AgentEvaluation) error
		ListByRun(ctx context.Context, runID uuid.UUID) ([]*AgentEvaluation, error)
	}
}

func NewStorage(pool *pgxpool.Pool) Storage {
	return Storage{
		Projects:         &ProjectStore{pool: pool},
		APIKeys:          &APIKeyStore{pool: pool},
		AITraces:         &AITraceStore{pool: pool},
		APIEvents:        &APIEventStore{pool: pool},
		InfraMetrics:     &InfraMetricStore{pool: pool},
		AgentRuns:        &AgentRunStore{pool: pool},
		AgentSteps:       &AgentStepStore{pool: pool},
		AgentTools:       &AgentToolStore{pool: pool},
		AgentEvaluations: &AgentEvaluationStore{pool: pool},
	}
}
