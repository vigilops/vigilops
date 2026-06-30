package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSession(t *testing.T, ctx context.Context, s Storage, userID [16]byte, hash []byte, expires time.Time) *Session {
	t.Helper()
	sess := &Session{UserID: userID, TokenHash: hash, ExpiresAt: expires}
	require.NoError(t, s.Sessions.Create(ctx, sess))
	return sess
}

func TestSessionStore_Create_andGetByHash(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "sess")

	sess := makeSession(t, ctx, s, u.ID, []byte("hash-live"), time.Now().Add(time.Hour))
	assert.False(t, sess.CreatedAt.IsZero())

	got, err := s.Sessions.GetByHash(ctx, []byte("hash-live"))
	require.NoError(t, err)
	assert.Equal(t, sess.ID, got.ID)
	assert.Equal(t, u.ID, got.UserID)
}

func TestSessionStore_GetByHash_treatsExpiredAsNotFound(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "expired")

	makeSession(t, ctx, s, u.ID, []byte("hash-expired"), time.Now().Add(-time.Minute))
	_, err := s.Sessions.GetByHash(ctx, []byte("hash-expired"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSessionStore_Delete_revokesSession(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "revoke")

	sess := makeSession(t, ctx, s, u.ID, []byte("hash-revoke"), time.Now().Add(time.Hour))
	require.NoError(t, s.Sessions.Delete(ctx, sess.ID))
	_, err := s.Sessions.GetByHash(ctx, []byte("hash-revoke"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSessionStore_DeleteByUser_revokesAll(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "logoutall")

	makeSession(t, ctx, s, u.ID, []byte("hash-a"), time.Now().Add(time.Hour))
	makeSession(t, ctx, s, u.ID, []byte("hash-b"), time.Now().Add(time.Hour))

	require.NoError(t, s.Sessions.DeleteByUser(ctx, u.ID))
	_, errA := s.Sessions.GetByHash(ctx, []byte("hash-a"))
	_, errB := s.Sessions.GetByHash(ctx, []byte("hash-b"))
	assert.ErrorIs(t, errA, ErrNotFound)
	assert.ErrorIs(t, errB, ErrNotFound)
}

func TestSessionStore_DeleteExpired_prunesOnlyExpired(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "prune")

	makeSession(t, ctx, s, u.ID, []byte("hash-old"), time.Now().Add(-time.Hour))
	live := makeSession(t, ctx, s, u.ID, []byte("hash-new"), time.Now().Add(time.Hour))

	_, err := s.Sessions.DeleteExpired(ctx)
	require.NoError(t, err)

	got, err := s.Sessions.GetByHash(ctx, []byte("hash-new"))
	require.NoError(t, err)
	assert.Equal(t, live.ID, got.ID)
}
