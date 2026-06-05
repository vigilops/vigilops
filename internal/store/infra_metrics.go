package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InfraMetric struct {
	Timestamp  time.Time `json:"timestamp"`
	ProjectID  uuid.UUID `json:"project_id"`
	Host       string    `json:"host"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Labels     []byte    `json:"labels,omitempty"`
}

type InfraMetricStore struct {
	pool *pgxpool.Pool
}

func (s *InfraMetricStore) Insert(ctx context.Context, m *InfraMetric) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}

	const q = `
		INSERT INTO infra_metrics (timestamp, project_id, host, metric_name, value, labels)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.pool.Exec(ctx, q,
		m.Timestamp, m.ProjectID, m.Host, m.MetricName, m.Value, m.Labels,
	)
	return err
}
