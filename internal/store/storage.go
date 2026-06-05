package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource already exists")
	QueryTimeoutDuration = 5 * time.Second
)

type Storage struct {
	Projects interface {
		Create(ctx context.Context, p *Project) error
		GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
		List(ctx context.Context) ([]*Project, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}
	APIKeys interface {
		Create(ctx context.Context, k *APIKey) error
		GetByHash(ctx context.Context, hash []byte) (*APIKey, error)
		TouchLastUsed(ctx context.Context, id uuid.UUID) error
		ListByProject(ctx context.Context, projectID uuid.UUID) ([]*APIKey, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}
}

func NewStorage(pool *pgxpool.Pool) Storage {
	return Storage{
		Projects: &ProjectStore{pool: pool},
		APIKeys:  &APIKeyStore{pool: pool},
	}
}
