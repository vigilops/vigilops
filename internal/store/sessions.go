package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Session struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	TokenHash  []byte     `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type SessionStore struct {
	pool *pgxpool.Pool
}

func (s *SessionStore) Create(ctx context.Context, sess *Session) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return s.pool.QueryRow(ctx, q, sess.UserID, sess.TokenHash, sess.ExpiresAt).
		Scan(&sess.ID, &sess.CreatedAt)
}

// GetByHash returns a live (unexpired) session for a token hash. Expired
// sessions are treated as not found.
func (s *SessionStore) GetByHash(ctx context.Context, hash []byte) (*Session, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, user_id, token_hash, created_at, expires_at, last_used_at
		FROM sessions
		WHERE token_hash = $1 AND expires_at > now()
	`
	sess := &Session{}
	err := s.pool.QueryRow(ctx, q, hash).Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash,
		&sess.CreatedAt, &sess.ExpiresAt, &sess.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sess, nil
}

func (s *SessionStore) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `UPDATE sessions SET last_used_at = now() WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, id)
	return err
}

// Delete revokes a single session (logout).
func (s *SessionStore) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `DELETE FROM sessions WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteByUser revokes every session for a user (logout everywhere / on
// password change or compromise).
func (s *SessionStore) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `DELETE FROM sessions WHERE user_id = $1`
	_, err := s.pool.Exec(ctx, q, userID)
	return err
}

// DeleteExpired prunes expired rows; run periodically.
func (s *SessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `DELETE FROM sessions WHERE expires_at <= now()`
	tag, err := s.pool.Exec(ctx, q)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
