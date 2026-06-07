// Package batch buffers rows in memory and flushes them in bulk via COPY.
// The generic core lives here; per-table flushers are in ingest.go.
package batch

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FlushFunc is invoked with a non-empty []T no longer than Config.MaxRows.
type FlushFunc[T any] func(ctx context.Context, rows []T) error

type Config struct {
	FlushInterval time.Duration
	MaxRows       int
	QueueDepth    int
}

func (c Config) withDefaults() Config {
	if c.FlushInterval <= 0 {
		c.FlushInterval = 500 * time.Millisecond
	}
	if c.MaxRows <= 0 {
		c.MaxRows = 500
	}
	if c.QueueDepth <= 0 {
		c.QueueDepth = 10_000
	}
	return c
}

// ErrBufferFull = client should retry; handlers surface this as 503 + Retry-After.
var ErrBufferFull = errors.New("batch buffer full")

type Buffer[T any] struct {
	name   string
	cfg    Config
	ch     chan T
	flush  FlushFunc[T]
	wg     sync.WaitGroup
	logger *zap.SugaredLogger
}

func New[T any](name string, cfg Config, logger *zap.SugaredLogger, flush FlushFunc[T]) *Buffer[T] {
	cfg = cfg.withDefaults()
	return &Buffer[T]{
		name:   name,
		cfg:    cfg,
		ch:     make(chan T, cfg.QueueDepth),
		flush:  flush,
		logger: logger,
	}
}

func (b *Buffer[T]) Start(ctx context.Context) {
	b.wg.Add(1)
	go b.run(ctx)
}

// Enqueue is non-blocking; returns ErrBufferFull instead of stalling.
func (b *Buffer[T]) Enqueue(_ context.Context, row T) error {
	select {
	case b.ch <- row:
		return nil
	default:
		return ErrBufferFull
	}
}

// Stop closes the queue and waits for the drain. Returns ctx.Err() if the
// shutdown deadline fires before the run loop exits.
func (b *Buffer[T]) Stop(ctx context.Context) error {
	close(b.ch)
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		b.logger.Warnw("batch stop deadline exceeded", "table", b.name)
		return ctx.Err()
	}
}

func (b *Buffer[T]) run(_ context.Context) {
	defer b.wg.Done()

	ticker := time.NewTicker(b.cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]T, 0, b.cfg.MaxRows)
	flushNow := func() {
		if len(batch) == 0 {
			return
		}
		rows := batch
		batch = make([]T, 0, b.cfg.MaxRows)
		// Use a fresh ctx — the rootCtx that fired SIGTERM is already
		// cancelled, and passing it to pgx.CopyFrom would abort the drain
		// mid-flight. The drain's wall-clock budget is enforced by the
		// ctx passed to Buffer.Stop.
		if err := b.flush(context.Background(), rows); err != nil {
			b.logger.Errorw("batch flush failed — rows dropped",
				"table", b.name, "rows", len(rows), "err", err)
		}
	}

	for {
		select {
		case row, ok := <-b.ch:
			if !ok {
				// Channel closed by Stop — drain remainder + final flush.
				for r := range b.ch {
					batch = append(batch, r)
					if len(batch) >= b.cfg.MaxRows {
						flushNow()
					}
				}
				flushNow()
				return
			}
			batch = append(batch, row)
			if len(batch) >= b.cfg.MaxRows {
				flushNow()
			}
		case <-ticker.C:
			flushNow()
		}
	}
}
