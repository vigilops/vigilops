package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeKey(t *testing.T, ctx context.Context, s Storage, projectID uuid.UUID, name string) *APIKey {
	t.Helper()
	k := &APIKey{
		ProjectID: projectID,
		KeyHash:   []byte("hash-" + name),
		Name:      name,
	}
	require.NoError(t, s.APIKeys.Create(ctx, k))
	return k
}

func TestAPIKeyStore_Create_assignsIDAndTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "keycreate")

	k := makeKey(t, ctx, s, p.ID, "primary")
	assert.NotEqual(t, uuid.Nil, k.ID)
	assert.False(t, k.CreatedAt.IsZero())
	assert.Nil(t, k.LastUsedAt)
}

func TestAPIKeyStore_GetByHash_returnsCreatedKey(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "keygethash")

	k := makeKey(t, ctx, s, p.ID, "lookup")
	got, err := s.APIKeys.GetByHash(ctx, k.KeyHash)
	require.NoError(t, err)
	assert.Equal(t, k.ID, got.ID)
	assert.Equal(t, p.ID, got.ProjectID)
}

func TestAPIKeyStore_GetByHash_notFoundForUnknownHash(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	_, err := s.APIKeys.GetByHash(ctx, []byte("nope"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestAPIKeyStore_TouchLastUsed_setsTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "keytouch")

	k := makeKey(t, ctx, s, p.ID, "touch")
	require.Nil(t, k.LastUsedAt)
	require.NoError(t, s.APIKeys.TouchLastUsed(ctx, k.ID))

	got, err := s.APIKeys.GetByHash(ctx, k.KeyHash)
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt)
}

func TestAPIKeyStore_ListByProject_returnsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "keys-empty")

	keys, err := s.APIKeys.ListByProject(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, keys)
	assert.Len(t, keys, 0)
}

func TestAPIKeyStore_ListByProject_orderedNewestFirst(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "keylist")

	makeKey(t, ctx, s, p.ID, "first")
	makeKey(t, ctx, s, p.ID, "second")

	keys, err := s.APIKeys.ListByProject(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.True(t, keys[0].CreatedAt.After(keys[1].CreatedAt) || keys[0].CreatedAt.Equal(keys[1].CreatedAt))
}

func TestAPIKeyStore_Delete_removesRow(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "keydelete")

	k := makeKey(t, ctx, s, p.ID, "deleteme")
	require.NoError(t, s.APIKeys.Delete(ctx, k.ID, p.ID))
	_, err := s.APIKeys.GetByHash(ctx, k.KeyHash)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestAPIKeyStore_Delete_notFoundForUnknownID(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	err := s.APIKeys.Delete(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}
