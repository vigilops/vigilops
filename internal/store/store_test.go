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
		addr = "postgres://keelwave:keelwave@localhost:5432/keelwave?sslmode=disable"
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

func testOrgForUser(t *testing.T, s Storage, u *User) *Organization {
	t.Helper()
	org := &Organization{Name: fmt.Sprintf("org-%d", time.Now().UnixNano())}
	require.NoError(t, s.Organizations.CreateWithOwner(context.Background(), org, u.ID))
	return org
}

// testProject creates an isolated project (with a throw-away owner) for one test.
// Cascade DELETE on the projects FK cleans up child rows when the test ends.
func testProject(t *testing.T, s Storage, label string) *Project {
	t.Helper()
	p, _ := testProjectWithOwner(t, s, label)
	return p
}

// testProjectWithOwner is like testProject but also returns the owning user.
func testProjectWithOwner(t *testing.T, s Storage, label string) (*Project, *User) {
	t.Helper()
	u := testUser(t, s, label)
	org := testOrgForUser(t, s, u)
	p := &Project{Name: fmt.Sprintf("test-%s-%d", label, time.Now().UnixNano())}
	require.NoError(t, s.Projects.Create(context.Background(), p, org.ID))
	t.Cleanup(func() {
		_ = s.Projects.Delete(context.Background(), p.ID)
	})
	return p, u
}
