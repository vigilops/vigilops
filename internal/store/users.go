package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Password   password   `json:"-"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	IsVerified bool       `json:"is_verified"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
}

type password struct {
	text *string
	hash []byte
}

func (p *password) Set(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p.text = &text
	p.hash = hash
	return nil
}

func (p *password) Compare(text string) bool {
	if len(p.hash) == 0 {
		return false
	}
	return bcrypt.CompareHashAndPassword(p.hash, []byte(text)) == nil
}

func (p *password) HasPassword() bool { return len(p.hash) > 0 }

type UserStore struct {
	pool          *pgxpool.Pool
	identities    *UserIdentityStore
	verifications *UserVerificationStore
}

func (s *UserStore) Create(ctx context.Context, u *User, tx pgx.Tx) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	if u.IsVerified && u.VerifiedAt == nil {
		now := time.Now()
		u.VerifiedAt = &now
	}

	const q = `
		INSERT INTO users (email, password, name, is_verified, verified_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, is_verified, verified_at
	`
	var row pgx.Row
	if tx != nil {
		row = tx.QueryRow(ctx, q, u.Email, u.Password.hash, u.Name, u.IsVerified, u.VerifiedAt)
	} else {
		row = s.pool.QueryRow(ctx, q, u.Email, u.Password.hash, u.Name, u.IsVerified, u.VerifiedAt)
	}
	err := row.Scan(&u.ID, &u.CreatedAt, &u.IsVerified, &u.VerifiedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, email, password, name, created_at, is_verified, verified_at
		FROM users
		WHERE id = $1
	`
	return scanUser(s.pool.QueryRow(ctx, q, id))
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	const q = `
		SELECT id, email, password, name, created_at, is_verified, verified_at
		FROM users
		WHERE email = $1
	`
	return scanUser(s.pool.QueryRow(ctx, q, email))
}

// Verify sets is_verified on the user and deletes the verification row atomically.
func (s *UserStore) Verify(ctx context.Context, tokenHash []byte) error {
	return withTx(s.pool, ctx, func(tx pgx.Tx) error {
		const q = `
			UPDATE users SET is_verified = true, verified_at = now()
			WHERE id = (
				SELECT user_id FROM user_verifications
				WHERE token = $1 AND expires_at > now()
			)
			RETURNING id
		`
		var id uuid.UUID
		if err := tx.QueryRow(ctx, q, tokenHash).Scan(&id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		_, err := tx.Exec(ctx, `DELETE FROM user_verifications WHERE user_id = $1`, id)
		return err
	})
}

// CreateWithVerification inserts a user, their password identity, and verification token atomically.
func (s *UserStore) CreateWithVerification(ctx context.Context, u *User, v *UserVerification) error {
	return withTx(s.pool, ctx, func(tx pgx.Tx) error {
		if err := s.Create(ctx, u, tx); err != nil {
			return err
		}
		v.UserID = u.ID
		if err := s.verifications.create(ctx, tx, v); err != nil {
			return err
		}
		return s.identities.create(ctx, tx, &UserIdentity{
			UserID:         u.ID,
			Provider:       "password",
			ProviderUserID: u.ID.String(),
			Email:          u.Email,
		})
	})
}

// CreateWithIdentity inserts a user and their OAuth identity atomically.
func (s *UserStore) CreateWithIdentity(ctx context.Context, u *User, identity *UserIdentity) error {
	return withTx(s.pool, ctx, func(tx pgx.Tx) error {
		if err := s.Create(ctx, u, tx); err != nil {
			return err
		}
		identity.UserID = u.ID
		return s.identities.create(ctx, tx, identity)
	})
}

func scanUser(row pgx.Row) (*User, error) {
	u := &User{}
	err := row.Scan(&u.ID, &u.Email, &u.Password.hash, &u.Name, &u.CreatedAt, &u.IsVerified, &u.VerifiedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}
