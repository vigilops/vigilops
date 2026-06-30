package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type OrganizationStore struct {
	pool    *pgxpool.Pool
	members *OrganizationMemberStore
}

var OwnerRole = "owner"

func (s *OrganizationStore) CreateWithOwner(ctx context.Context, a *Organization, ownerUserID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return withTx(s.pool, ctx, func(tx pgx.Tx) error {
		if err := s.create(ctx, tx, a); err != nil {
			return err
		}
		return s.members.add(ctx, tx, a.ID, ownerUserID, OwnerRole)
	})
}

func (s *OrganizationStore) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `SELECT id, name, created_at FROM organizations WHERE id = $1`
	a := &Organization{}
	err := s.pool.QueryRow(ctx, q, id).Scan(&a.ID, &a.Name, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

func (s *OrganizationStore) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT a.id, a.name, a.created_at
		FROM organizations a
		JOIN organization_members m ON m.organization_id = a.id
		WHERE m.user_id = $1
		ORDER BY a.created_at
	`
	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	organizations := make([]*Organization, 0)
	for rows.Next() {
		a := &Organization{}
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt); err != nil {
			return nil, err
		}
		organizations = append(organizations, a)
	}
	return organizations, rows.Err()
}

func (s *OrganizationStore) Update(ctx context.Context, id uuid.UUID, name string) (*Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `UPDATE organizations SET name = $2 WHERE id = $1 RETURNING id, name, created_at`
	a := &Organization{}
	err := s.pool.QueryRow(ctx, q, id, name).Scan(&a.ID, &a.Name, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

func (s *OrganizationStore) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `DELETE FROM organizations WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *OrganizationStore) create(ctx context.Context, tx pgx.Tx, a *Organization) error {
	const q = `INSERT INTO organizations (name) VALUES ($1) RETURNING id, created_at`
	return tx.QueryRow(ctx, q, a.Name).Scan(&a.ID, &a.CreatedAt)
}
