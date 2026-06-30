package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Project struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	OrganizationID uuid.UUID `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type ProjectStore struct {
	pool *pgxpool.Pool
}

func (s *ProjectStore) Create(ctx context.Context, p *Project, orgID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		INSERT INTO projects (name, organization_id)
		VALUES ($1, $2)
		RETURNING id, created_at
	`
	p.OrganizationID = orgID
	return s.pool.QueryRow(ctx, q, p.Name, orgID).Scan(&p.ID, &p.CreatedAt)
}

func (s *ProjectStore) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*Project, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT p.id, p.name, p.organization_id, p.created_at
		FROM projects p
		JOIN organization_members om ON om.organization_id = p.organization_id
		WHERE p.id = $1 AND om.user_id = $2
	`
	p := &Project{}
	err := s.pool.QueryRow(ctx, q, id, userID).
		Scan(&p.ID, &p.Name, &p.OrganizationID, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *ProjectStore) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `SELECT id, name, organization_id, created_at FROM projects WHERE id = $1`
	p := &Project{}
	err := s.pool.QueryRow(ctx, q, id).Scan(&p.ID, &p.Name, &p.OrganizationID, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *ProjectStore) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*Project, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, name, organization_id, created_at
		FROM projects
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*Project, 0)
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.OrganizationID, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *ProjectStore) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Project, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT p.id, p.name, p.organization_id, p.created_at
		FROM projects p
		JOIN organization_members om ON om.organization_id = p.organization_id
		WHERE om.user_id = $1
		ORDER BY p.created_at DESC
	`
	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*Project, 0)
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.OrganizationID, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *ProjectStore) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `DELETE FROM projects WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
