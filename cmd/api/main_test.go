package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/yusufnuru/vigil/internal/auth"
	"github.com/yusufnuru/vigil/internal/batch"
	"github.com/yusufnuru/vigil/internal/store"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	addr := os.Getenv("TEST_DB_ADDR")
	if addr == "" {
		addr = "postgres://vigil:vigil@localhost:5432/vigil?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), addr)
	if err != nil {
		log.Fatalf("test db connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("test db ping (run `make db-up && make migrate-up`): %v", err)
	}
	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func newTestServer(t *testing.T) (srv *httptest.Server, projectID string, apiKey string, app *application) {
	t.Helper()
	ctx := context.Background()

	s := store.NewStorage(testPool)

	proj := &store.Project{Name: fmt.Sprintf("test-%d", time.Now().UnixNano())}
	require.NoError(t, s.Projects.Create(ctx, proj))

	plaintext, hash, err := auth.Generate()
	require.NoError(t, err)
	key := &store.APIKey{ProjectID: proj.ID, KeyHash: hash, Name: "test"}
	require.NoError(t, s.APIKeys.Create(ctx, key))

	app = &application{
		config: config{
			env: "test",
			rateLimit: rateLimitConfig{
				ingestIPPerMinute:  1_000_000,
				ingestKeyPerMinute: 1_000_000,
				ingestWindow:       time.Minute,
			},
		},
		pool:     testPool,
		store:    s,
		batchers: batch.NewBatchers(testPool, batch.Config{}, zap.NewNop().Sugar()),
		logger:   zap.NewNop().Sugar(),
	}

	srv = httptest.NewServer(app.mount())
	t.Cleanup(func() {
		srv.Close()
		_ = s.Projects.Delete(ctx, proj.ID)
	})
	return srv, proj.ID.String(), plaintext, app
}

// doJSON issues an authenticated request and unmarshals the response body.
func doJSON(t *testing.T, method, url, key string, body any, out any) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = nopReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	require.NoError(t, err)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	if out != nil && len(raw) > 0 {
		require.NoError(t, json.Unmarshal(raw, out), "body=%s", raw)
	}
	return resp, raw
}

type nopReader []byte

func (b nopReader) Read(p []byte) (int, error) {
	n := copy(p, b)
	if n < len(b) {
		return n, nil
	}
	return n, io.EOF
}
