package store

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// testPool is shared across all tests in the package. Constructed once via
// TestMain so we don't pay connection-pool startup cost per test.
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
		log.Fatalf("test db ping (is `make db-up && make migrate-up` run?): %v", err)
	}
	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func testStorage(t *testing.T) Storage {
	t.Helper()
	return NewStorage(testPool)
}

// testProject creates an isolated project for one test. The cascade DELETE on
// the projects FK cleans up every child row when the test ends.
func testProject(t *testing.T, s Storage, label string) *Project {
	t.Helper()
	p := &Project{Name: fmt.Sprintf("test-%s-%d", label, time.Now().UnixNano())}
	require.NoError(t, s.Projects.Create(context.Background(), p))
	t.Cleanup(func() {
		_ = s.Projects.Delete(context.Background(), p.ID)
	})
	return p
}
