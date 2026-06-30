package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationInvite struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	Token          []byte     `json:"-"`
	ExpiresAt      time.Time  `json:"expires_at"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type OrganizationInviteStore struct {
	pool    *pgxpool.Pool
	members *OrganizationMemberStore
}

func (s *OrganizationInviteStore) Create(ctx context.Context, inv *OrganizationInvite) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		INSERT INTO organization_invites (organization_id, email, role, token, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return s.pool.QueryRow(ctx, q,
		inv.OrganizationID, inv.Email, inv.Role, inv.Token, inv.ExpiresAt,
	).Scan(&inv.ID, &inv.CreatedAt)
}

func (s *OrganizationInviteStore) GetByToken(ctx context.Context, hash []byte) (*OrganizationInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, organization_id, email, role, token, expires_at, accepted_at, created_at
		FROM organization_invites
		WHERE token = $1
	`
	inv := &OrganizationInvite{}
	err := s.pool.QueryRow(ctx, q, hash).Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role,
		&inv.Token, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return inv, nil
}

func (s *OrganizationInviteStore) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*OrganizationInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, organization_id, email, role, expires_at, accepted_at, created_at
		FROM organization_invites
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*OrganizationInvite, 0)
	for rows.Next() {
		inv := &OrganizationInvite{}
		if err := rows.Scan(
			&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role,
			&inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// Accept marks the invite accepted and adds the user as a member in one transaction.
func (s *OrganizationInviteStore) Accept(ctx context.Context, inviteID, userID uuid.UUID, role string) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return withTx(s.pool, ctx, func(tx pgx.Tx) error {
		orgID, err := s.markAccepted(ctx, tx, inviteID)
		if err != nil {
			return err
		}
		return s.members.add(ctx, tx, orgID, userID, role)
	})
}

func (s *OrganizationInviteStore) markAccepted(ctx context.Context, tx pgx.Tx, inviteID uuid.UUID) (uuid.UUID, error) {
	const q = `
		UPDATE organization_invites
		SET accepted_at = now()
		WHERE id = $1 AND accepted_at IS NULL
		RETURNING organization_id
	`
	var orgID uuid.UUID
	err := tx.QueryRow(ctx, q, inviteID).Scan(&orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrConflict
		}
		return uuid.Nil, err
	}
	return orgID, nil
}

func (s *OrganizationInviteStore) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	tag, err := s.pool.Exec(ctx, `DELETE FROM organization_invites WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
