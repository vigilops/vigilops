package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testUser creates an isolated user and cleans it up when the test ends.
func testUser(t *testing.T, s Storage, label string) *User {
	t.Helper()
	u := &User{
		Email: fmt.Sprintf("%s-%d@test.local", label, time.Now().UnixNano()),
		Name:  label,
	}
	require.NoError(t, u.Password.Set("password-"+label))
	require.NoError(t, s.Users.Create(context.Background(), u, nil))
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.ID)
	})
	return u
}

func TestUserStore_Create_assignsIDAndTimestamp(t *testing.T) {
	s := testStorage(t)
	u := testUser(t, s, "create")
	assert.NotEqual(t, uuid.Nil, u.ID)
	assert.False(t, u.CreatedAt.IsZero())
}

func TestUserStore_GetByEmail_isCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "CaseTest")

	got, err := s.Users.GetByEmail(ctx, u.Email)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	// citext column → lookup ignores case.
	upper, err := s.Users.GetByEmail(ctx, toUpper(u.Email))
	require.NoError(t, err)
	assert.Equal(t, u.ID, upper.ID)
}

func TestUserStore_Create_rejectsDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "dup")

	dup := &User{Email: u.Email, Name: "dup2"}
	require.NoError(t, dup.Password.Set("whatever123"))
	err := s.Users.Create(ctx, dup, nil)
	assert.ErrorIs(t, err, ErrConflict)
}

func TestPassword_SetAndCompare(t *testing.T) {
	var p password
	require.NoError(t, p.Set("correct horse battery staple"))
	assert.True(t, p.HasPassword())
	assert.True(t, p.Compare("correct horse battery staple"))
	assert.False(t, p.Compare("wrong"))
}

func TestPassword_ZeroValueNeverMatches(t *testing.T) {
	var p password // OAuth-only user: no hash set
	assert.False(t, p.HasPassword())
	assert.False(t, p.Compare(""))
	assert.False(t, p.Compare("anything"))
}

func TestUserStore_Create_persistsHashThroughRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "roundtrip")

	got, err := s.Users.GetByEmail(ctx, u.Email)
	require.NoError(t, err)
	assert.True(t, got.Password.Compare("password-roundtrip"), "hash must survive store round-trip")
}

func TestUserStore_GetByID_notFound(t *testing.T) {
	s := testStorage(t)
	_, err := s.Users.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func toUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
