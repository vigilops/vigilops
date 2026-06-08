package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectStore_Create_assignsIDAndTimestamp(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)

	p := &Project{Name: "create-test"}
	require.NoError(t, s.Projects.Create(ctx, p))

	assert.NotEqual(t, uuid.Nil, p.ID)
	assert.False(t, p.CreatedAt.IsZero())
	t.Cleanup(func() { _ = s.Projects.Delete(ctx, p.ID) })
}

func TestProjectStore_GetByID_returnsCreatedRow(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "getbyid")

	got, err := s.Projects.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, p.Name, got.Name)
}

func TestProjectStore_GetByID_notFoundForUnknownID(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	_, err := s.Projects.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProjectStore_List_includesCreatedRow(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p := testProject(t, s, "list")

	all, err := s.Projects.List(ctx)
	require.NoError(t, err)

	var found bool
	for _, x := range all {
		if x.ID == p.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "project should appear in List")
}

func TestProjectStore_Delete_removesRow(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)

	p := &Project{Name: "delete-test"}
	require.NoError(t, s.Projects.Create(ctx, p))
	require.NoError(t, s.Projects.Delete(ctx, p.ID))

	_, err := s.Projects.GetByID(ctx, p.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProjectStore_Delete_notFoundForUnknownID(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	err := s.Projects.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}
