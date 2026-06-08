package batch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func nopLogger() *zap.SugaredLogger { return zap.NewNop().Sugar() }

func TestNew_appliesDefaults(t *testing.T) {
	b := New("t", Config{}, nopLogger(), func(ctx context.Context, rows []int) error { return nil })
	assert.Equal(t, 500*time.Millisecond, b.cfg.FlushInterval)
	assert.Equal(t, 500, b.cfg.MaxRows)
	assert.Equal(t, 10_000, b.cfg.QueueDepth)
}

func TestEnqueue_returnsErrBufferFullWhenChannelFull(t *testing.T) {
	b := New("t", Config{QueueDepth: 2, FlushInterval: time.Hour, MaxRows: 1000}, nopLogger(),
		func(ctx context.Context, rows []int) error { return nil })
	// Don't Start — fills channel without drain.
	require.NoError(t, b.Enqueue(context.Background(), 1))
	require.NoError(t, b.Enqueue(context.Background(), 2))
	err := b.Enqueue(context.Background(), 3)
	assert.ErrorIs(t, err, ErrBufferFull)
}

func TestFlushFiresOnMaxRows(t *testing.T) {
	var flushes atomic.Int64
	var got [][]int
	var mu sync.Mutex
	b := New("t", Config{QueueDepth: 100, FlushInterval: time.Hour, MaxRows: 3}, nopLogger(),
		func(ctx context.Context, rows []int) error {
			mu.Lock()
			got = append(got, append([]int(nil), rows...))
			mu.Unlock()
			flushes.Add(1)
			return nil
		})
	b.Start(context.Background())
	for i := range 6 {
		require.NoError(t, b.Enqueue(context.Background(), i))
	}
	require.NoError(t, b.Stop(context.Background()))

	mu.Lock()
	defer mu.Unlock()
	total := 0
	for _, batch := range got {
		total += len(batch)
	}
	assert.Equal(t, 6, total, "all rows must reach flush callback")
	assert.GreaterOrEqual(t, int64(2), flushes.Load(), "at least 2 flushes (max-rows + drain)")
}

func TestFlushFiresOnTickerWhenBelowMaxRows(t *testing.T) {
	var seen atomic.Int64
	b := New("t", Config{QueueDepth: 100, FlushInterval: 50 * time.Millisecond, MaxRows: 1000}, nopLogger(),
		func(ctx context.Context, rows []int) error { seen.Add(int64(len(rows))); return nil })
	b.Start(context.Background())
	require.NoError(t, b.Enqueue(context.Background(), 1))
	require.Eventually(t, func() bool { return seen.Load() == 1 }, time.Second, 5*time.Millisecond)
	require.NoError(t, b.Stop(context.Background()))
}

func TestStop_drainsRemainingRows(t *testing.T) {
	var seen atomic.Int64
	b := New("t", Config{QueueDepth: 100, FlushInterval: time.Hour, MaxRows: 100}, nopLogger(),
		func(ctx context.Context, rows []int) error { seen.Add(int64(len(rows))); return nil })
	b.Start(context.Background())
	for i := range 50 {
		require.NoError(t, b.Enqueue(context.Background(), i))
	}
	require.NoError(t, b.Stop(context.Background()))
	assert.Equal(t, int64(50), seen.Load(), "Stop must drain all queued rows")
}

func TestStop_flushUsesBackgroundCtxAfterParentCancel(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	var ctxErr atomic.Value
	b := New("t", Config{QueueDepth: 100, FlushInterval: time.Hour, MaxRows: 100}, nopLogger(),
		func(ctx context.Context, rows []int) error {
			if err := ctx.Err(); err != nil {
				ctxErr.Store(err)
			}
			return nil
		})
	b.Start(parent)

	require.NoError(t, b.Enqueue(parent, 1))
	cancel() // simulate SIGTERM cancelling rootCtx

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutCancel()
	require.NoError(t, b.Stop(shutCtx))

	assert.Nil(t, ctxErr.Load(), "flush ctx must be background, NOT the cancelled parent")
}

func TestStop_returnsCtxErrIfDeadlineFires(t *testing.T) {
	hang := make(chan struct{})
	b := New("t", Config{QueueDepth: 100, FlushInterval: time.Hour, MaxRows: 1}, nopLogger(),
		func(ctx context.Context, rows []int) error { <-hang; return nil })
	b.Start(context.Background())
	require.NoError(t, b.Enqueue(context.Background(), 1))

	shutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := b.Stop(shutCtx)
	close(hang)
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "Stop should surface shutCtx deadline")
}

func TestEnqueue_concurrent_lossLessUnderRaceDetector(t *testing.T) {
	const workers = 50
	const perWorker = 200
	var seen atomic.Int64
	b := New("t", Config{QueueDepth: workers * perWorker, FlushInterval: 10 * time.Millisecond, MaxRows: 100}, nopLogger(),
		func(ctx context.Context, rows []int) error { seen.Add(int64(len(rows))); return nil })
	b.Start(context.Background())

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for i := range perWorker {
				require.NoError(t, b.Enqueue(context.Background(), i))
			}
		})
	}
	wg.Wait()
	require.NoError(t, b.Stop(context.Background()))

	assert.Equal(t, int64(workers*perWorker), seen.Load(), "every enqueued row must reach the flush callback")
}
