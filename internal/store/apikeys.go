package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKey struct {
	ID         uuid.UUID  `json:"id"`
	ProjectID  uuid.UUID  `json:"project_id"`
	KeyHash    []byte     `json:"-"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type APIKeyStore struct {
	pool *pgxpool.Pool
}

func (s *APIKeyStore) Create(ctx context.Context, k *APIKey) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		INSERT INTO api_keys (project_id, key_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return s.pool.QueryRow(ctx, q, k.ProjectID, k.KeyHash, k.Name).
		Scan(&k.ID, &k.CreatedAt)
}

func (s *APIKeyStore) GetByHash(ctx context.Context, hash []byte) (*APIKey, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, project_id, key_hash, name, created_at, last_used_at
		FROM api_keys
		WHERE key_hash = $1
	`
	k := &APIKey{}
	err := s.pool.QueryRow(ctx, q, hash).Scan(
		&k.ID, &k.ProjectID, &k.KeyHash, &k.Name, &k.CreatedAt, &k.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return k, nil
}

func (s *APIKeyStore) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `UPDATE api_keys SET last_used_at = now() WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, id)
	return err
}

func (s *APIKeyStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*APIKey, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, project_id, key_hash, name, created_at, last_used_at
		FROM api_keys
		WHERE project_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		k := &APIKey{}
		if err := rows.Scan(
			&k.ID, &k.ProjectID, &k.KeyHash, &k.Name, &k.CreatedAt, &k.LastUsedAt,
		); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *APIKeyStore) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `DELETE FROM api_keys WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
