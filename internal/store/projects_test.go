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
	u := testUser(t, s, "create")
	org := testOrgForUser(t, s, u)

	p := &Project{Name: "create-test"}
	require.NoError(t, s.Projects.Create(ctx, p, org.ID))

	assert.NotEqual(t, uuid.Nil, p.ID)
	assert.False(t, p.CreatedAt.IsZero())
	assert.Equal(t, org.ID, p.OrganizationID)
	t.Cleanup(func() { _ = s.Projects.Delete(ctx, p.ID) })
}

func TestProjectStore_GetByIDForUser_returnsCreatedRow(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p, u := testProjectWithOwner(t, s, "getbyid")

	got, err := s.Projects.GetByIDForUser(ctx, p.ID, u.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, p.Name, got.Name)
}

func TestProjectStore_GetByIDForUser_notFoundForUnknownID(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "notfound")

	_, err := s.Projects.GetByIDForUser(ctx, uuid.New(), u.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProjectStore_GetByIDForUser_notFoundForWrongUser(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p, _ := testProjectWithOwner(t, s, "wrong-user")
	other := testUser(t, s, "wrong")

	_, err := s.Projects.GetByIDForUser(ctx, p.ID, other.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProjectStore_ListByUser_includesCreatedRow(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	p, u := testProjectWithOwner(t, s, "list")

	all, err := s.Projects.ListByUser(ctx, u.ID)
	require.NoError(t, err)

	var found bool
	for _, x := range all {
		if x.ID == p.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "project should appear in ListByUser")
}

func TestProjectStore_ListByUser_emptyForOtherUser(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	_, _ = testProjectWithOwner(t, s, "list-other")
	other := testUser(t, s, "other")

	all, err := s.Projects.ListByUser(ctx, other.ID)
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestProjectStore_Delete_removesRow(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "delete")
	org := testOrgForUser(t, s, u)

	p := &Project{Name: "delete-test"}
	require.NoError(t, s.Projects.Create(ctx, p, org.ID))
	require.NoError(t, s.Projects.Delete(ctx, p.ID))

	_, err := s.Projects.GetByIDForUser(ctx, p.ID, u.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProjectStore_Delete_notFoundForUnknownID(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	err := s.Projects.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}
