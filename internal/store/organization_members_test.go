package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addMember(t *testing.T, orgID, userID uuid.UUID, role string) {
	t.Helper()
	ms := &OrganizationMemberStore{pool: testPool}
	err := withTx(testPool, context.Background(), func(tx pgx.Tx) error {
		return ms.add(context.Background(), tx, orgID, userID, role)
	})
	require.NoError(t, err)
}

func TestOrganizationMemberStore_Add_andGet(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	owner := testUser(t, s, "mowner")
	a := testOrganization(t, s, owner.ID, "members")
	member := testUser(t, s, "member")

	addMember(t, a.ID, member.ID, "member")

	got, err := s.OrganizationMembers.Get(ctx, a.ID, member.ID)
	require.NoError(t, err)
	assert.Equal(t, "member", got.Role)
}

func TestOrganizationMemberStore_Get_notFoundForNonMember(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	owner := testUser(t, s, "nmowner")
	a := testOrganization(t, s, owner.ID, "nm")
	stranger := testUser(t, s, "stranger")

	_, err := s.OrganizationMembers.Get(ctx, a.ID, stranger.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestOrganizationMemberStore_Add_rejectsDuplicate(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	s := testStorage(t)
	owner := testUser(t, s, "dupowner")
	a := testOrganization(t, s, owner.ID, "dupm")

	ms := &OrganizationMemberStore{pool: testPool}
	err := withTx(testPool, context.Background(), func(tx pgx.Tx) error {
		return ms.add(context.Background(), tx, a.ID, owner.ID, "admin")
	})
	assert.ErrorIs(t, err, ErrConflict)
}

func TestOrganizationMemberStore_UpdateRole(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	owner := testUser(t, s, "urowner")
	a := testOrganization(t, s, owner.ID, "ur")
	u := testUser(t, s, "promote")
	addMember(t, a.ID, u.ID, "member")

	require.NoError(t, s.OrganizationMembers.UpdateRole(ctx, a.ID, u.ID, "admin"))
	got, err := s.OrganizationMembers.Get(ctx, a.ID, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "admin", got.Role)
}

func TestOrganizationMemberStore_Remove(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	owner := testUser(t, s, "rmowner")
	a := testOrganization(t, s, owner.ID, "rm")
	u := testUser(t, s, "removee")
	addMember(t, a.ID, u.ID, "member")

	require.NoError(t, s.OrganizationMembers.Remove(ctx, a.ID, u.ID))
	_, err := s.OrganizationMembers.Get(ctx, a.ID, u.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestOrganizationMemberStore_ListByOrganization_includesUserIdentity(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	owner := testUser(t, s, "listowner")
	a := testOrganization(t, s, owner.ID, "list")

	members, err := s.OrganizationMembers.ListByOrganization(ctx, a.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, owner.ID, members[0].UserID)
	assert.Equal(t, owner.Email, members[0].Email)
	assert.Equal(t, "owner", members[0].Role)
}
