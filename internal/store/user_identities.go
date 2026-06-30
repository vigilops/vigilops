package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserIdentity struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	Email          string    `json:"email"`
	CreatedAt      time.Time `json:"created_at"`
}

type UserIdentityStore struct {
	pool *pgxpool.Pool
}

func (s *UserIdentityStore) Create(ctx context.Context, i *UserIdentity) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		INSERT INTO user_identities (user_id, provider, provider_user_id, email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := s.pool.QueryRow(ctx, q, i.UserID, i.Provider, i.ProviderUserID, i.Email).
		Scan(&i.ID, &i.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *UserIdentityStore) create(ctx context.Context, tx pgx.Tx, i *UserIdentity) error {
	const q = `
		INSERT INTO user_identities (user_id, provider, provider_user_id, email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := tx.QueryRow(ctx, q, i.UserID, i.Provider, i.ProviderUserID, i.Email).
		Scan(&i.ID, &i.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

// GetByProvider looks up an identity by (provider, provider_user_id) — the key
// the OAuth callback uses to recognise a returning user.
func (s *UserIdentityStore) GetByProvider(ctx context.Context, provider, providerUserID string) (*UserIdentity, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, user_id, provider, provider_user_id, email, created_at
		FROM user_identities
		WHERE provider = $1 AND provider_user_id = $2
	`
	i := &UserIdentity{}
	err := s.pool.QueryRow(ctx, q, provider, providerUserID).Scan(
		&i.ID, &i.UserID, &i.Provider, &i.ProviderUserID, &i.Email, &i.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return i, nil
}
