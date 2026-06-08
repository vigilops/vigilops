package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfraMetricStore_Insert_assignsTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "metric")

	m := &InfraMetric{
		ProjectID:  p.ID,
		Host:       "web-1",
		MetricName: "cpu_percent",
		Value:      42.5,
		Labels:     []byte(`{"region":"us-east"}`),
	}
	require.NoError(t, s.InfraMetrics.Insert(ctx, m))
	assert.False(t, m.Timestamp.IsZero())
}
