package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationMember struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

// MemberWithUser pairs membership with the user's identity for member lists.
type MemberWithUser struct {
	OrganizationMember
	Email string `json:"email"`
	Name  string `json:"name"`
}

type OrganizationMemberStore struct {
	pool *pgxpool.Pool
}

func (s *OrganizationMemberStore) Get(ctx context.Context, organizationID, userID uuid.UUID) (*OrganizationMember, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT organization_id, user_id, role, created_at
		FROM organization_members
		WHERE organization_id = $1 AND user_id = $2
	`
	m := &OrganizationMember{}
	err := s.pool.QueryRow(ctx, q, organizationID, userID).
		Scan(&m.OrganizationID, &m.UserID, &m.Role, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (s *OrganizationMemberStore) ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]*MemberWithUser, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT m.organization_id, m.user_id, m.role, m.created_at, u.email, u.name
		FROM organization_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1
		ORDER BY m.created_at
	`
	rows, err := s.pool.Query(ctx, q, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]*MemberWithUser, 0)
	for rows.Next() {
		m := &MemberWithUser{}
		if err := rows.Scan(
			&m.OrganizationID, &m.UserID, &m.Role, &m.CreatedAt, &m.Email, &m.Name,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *OrganizationMemberStore) UpdateRole(ctx context.Context, organizationID, userID uuid.UUID, role string) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		UPDATE organization_members SET role = $3
		WHERE organization_id = $1 AND user_id = $2
	`
	tag, err := s.pool.Exec(ctx, q, organizationID, userID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *OrganizationMemberStore) Remove(ctx context.Context, organizationID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `DELETE FROM organization_members WHERE organization_id = $1 AND user_id = $2`
	tag, err := s.pool.Exec(ctx, q, organizationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *OrganizationMemberStore) add(ctx context.Context, tx pgx.Tx, orgID, userID uuid.UUID, role string) error {
	const q = `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, $3)
	`
	_, err := tx.Exec(ctx, q, orgID, userID, role)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}
