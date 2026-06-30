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

// testOrganization creates an organization owned by ownerID and cleans it up after.
func testOrganization(t *testing.T, s Storage, ownerID uuid.UUID, label string) *Organization {
	t.Helper()
	a := &Organization{Name: fmt.Sprintf("acct-%s-%d", label, time.Now().UnixNano())}
	require.NoError(t, s.Organizations.CreateWithOwner(context.Background(), a, ownerID))
	t.Cleanup(func() {
		_ = s.Organizations.Delete(context.Background(), a.ID)
	})
	return a
}

func TestOrganizationStore_CreateWithOwner_addsOwnerMembership(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	owner := testUser(t, s, "owner")

	a := testOrganization(t, s, owner.ID, "withowner")
	assert.NotEqual(t, uuid.Nil, a.ID)

	m, err := s.OrganizationMembers.Get(ctx, a.ID, owner.ID)
	require.NoError(t, err)
	assert.Equal(t, "owner", m.Role)
}

func TestOrganizationStore_ListByUser_returnsMemberships(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	u := testUser(t, s, "lister")

	a1 := testOrganization(t, s, u.ID, "one")
	a2 := testOrganization(t, s, u.ID, "two")

	got, err := s.Organizations.ListByUser(ctx, u.ID)
	require.NoError(t, err)

	ids := map[uuid.UUID]bool{}
	for _, a := range got {
		ids[a.ID] = true
	}
	assert.True(t, ids[a1.ID])
	assert.True(t, ids[a2.ID])
}

func TestOrganizationStore_ListByUser_excludesNonMemberOrganizations(t *testing.T) {
	ctx := context.Background()
	s := testStorage(t)
	owner := testUser(t, s, "haver")
	outsider := testUser(t, s, "outsider")

	a := testOrganization(t, s, owner.ID, "private")

	got, err := s.Organizations.ListByUser(ctx, outsider.ID)
	require.NoError(t, err)
	for _, acc := range got {
		assert.NotEqual(t, a.ID, acc.ID, "outsider must not see owner's organization")
	}
}
