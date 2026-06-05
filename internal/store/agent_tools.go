package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentTool struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	ToolName    string    `json:"tool_name"`
	Description *string   `json:"description,omitempty"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type AgentToolStore struct {
	pool *pgxpool.Pool
}

// UpsertSeen records that a tool was seen — called from the agent step write
// path. Inserts on first sight, bumps last_seen_at otherwise. Lets the tool
// registry auto-populate without explicit registration.
func (s *AgentToolStore) UpsertSeen(ctx context.Context, projectID uuid.UUID, toolName string) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		INSERT INTO agent_tools (project_id, tool_name)
		VALUES ($1, $2)
		ON CONFLICT (project_id, tool_name)
		DO UPDATE SET last_seen_at = now()
	`
	_, err := s.pool.Exec(ctx, q, projectID, toolName)
	return err
}

func (s *AgentToolStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*AgentTool, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, project_id, tool_name, description, first_seen_at, last_seen_at
		FROM agent_tools
		WHERE project_id = $1
		ORDER BY last_seen_at DESC
	`
	rows, err := s.pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*AgentTool
	for rows.Next() {
		t := &AgentTool{}
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.ToolName, &t.Description,
			&t.FirstSeenAt, &t.LastSeenAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
