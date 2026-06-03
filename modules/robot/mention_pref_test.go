package robot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDecideOwnership covers the creator-ownership decision used by the
// owner-scoped mention_pref endpoints (octo-server#237). The ownership reject
// paths (404 not-found, 403 forbidden) are the acceptance-required coverage.
func TestDecideOwnership(t *testing.T) {
	cases := []struct {
		name       string
		creatorUID string
		loginUID   string
		want       ownershipResult
	}{
		{"creator matches → OK", "owner_1", "owner_1", ownershipOK},
		{"robot missing (empty creator) → 404", "", "owner_1", ownershipNotFound},
		{"empty creator + empty login → 404", "", "", ownershipNotFound},
		{"different user → 403", "owner_1", "intruder_2", ownershipForbidden},
		{"creator set, login empty → 403", "owner_1", "", ownershipForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, decideOwnership(tc.creatorUID, tc.loginUID))
		})
	}
}

// TestClampGroupsLimit verifies default (30), upper cap (100), and floor handling.
func TestClampGroupsLimit(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", groupsListDefaultLimit},
		{"abc", groupsListDefaultLimit},
		{"0", groupsListDefaultLimit},
		{"-5", groupsListDefaultLimit},
		{"15", 15},
		{"30", 30},
		{"100", 100},
		{"101", groupsListMaxLimit},
		{"99999", groupsListMaxLimit},
		{"  20  ", 20},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			assert.Equal(t, tc.want, clampGroupsLimit(tc.raw))
		})
	}
}

// TestGroupsCursorRoundTrip verifies the opaque cursor encodes/decodes and that
// blank/garbage decodes to 0 (first page) — keeping the cursor opaque to clients.
func TestGroupsCursorRoundTrip(t *testing.T) {
	for _, id := range []int64{1, 42, 1_000_000, 9_223_372_036_854_775_807} {
		enc := encodeGroupsCursor(id)
		assert.NotEqual(t, "", enc)
		assert.Equal(t, id, decodeGroupsCursor(enc))
	}

	// Blank and non-base64 / non-numeric inputs fall back to 0 (first page).
	assert.Equal(t, int64(0), decodeGroupsCursor(""))
	assert.Equal(t, int64(0), decodeGroupsCursor("   "))
	assert.Equal(t, int64(0), decodeGroupsCursor("!!!not-base64!!!"))
	// Valid base64 of a non-numeric string also falls back to 0.
	assert.Equal(t, int64(0), decodeGroupsCursor("YWJj")) // base64("abc")
}
