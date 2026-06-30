package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserVerification struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Token     []byte    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type UserVerificationStore struct {
	pool *pgxpool.Pool
}

func (s *UserVerificationStore) Create(ctx context.Context, v *UserVerification) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		INSERT INTO user_verifications (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return s.pool.QueryRow(ctx, q, v.UserID, v.Token, v.ExpiresAt).
		Scan(&v.ID, &v.CreatedAt)
}

func (s *UserVerificationStore) create(ctx context.Context, tx pgx.Tx, v *UserVerification) error {
	const q = `
		INSERT INTO user_verifications (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return tx.QueryRow(ctx, q, v.UserID, v.Token, v.ExpiresAt).
		Scan(&v.ID, &v.CreatedAt)
}

func (s *UserVerificationStore) GetByToken(ctx context.Context, hash []byte) (*UserVerification, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, user_id, token, expires_at, created_at
		FROM user_verifications
		WHERE token = $1
	`
	v := &UserVerification{}
	err := s.pool.QueryRow(ctx, q, hash).Scan(
		&v.ID, &v.UserID, &v.Token, &v.ExpiresAt, &v.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return v, nil
}

func (s *UserVerificationStore) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.pool.Exec(ctx, `DELETE FROM user_verifications WHERE user_id = $1`, userID)
	return err
}
