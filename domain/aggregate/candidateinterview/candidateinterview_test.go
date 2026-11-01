package candidateinterview_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureNow returns a reproducible UTC millisecond-precision instant.
func fixtureNow() time.Time { return time.Date(2026, 11, 2, 8, 5, 0, 123000000, time.UTC) }

// mustVersion parses a notice version or fails the test.
func mustVersion(t *testing.T, raw string) candidateinterview.NoticeVersion {
	t.Helper()
	v, err := candidateinterview.ParseNoticeVersion(raw)
	require.NoError(t, err)
	return v
}

// mustController parses a controller id or fails the test.
func mustController(t *testing.T, raw string) candidateinterview.ControllerID {
	t.Helper()
	c, err := candidateinterview.ParseControllerID(raw)
	require.NoError(t, err)
	return c
}

// TestParseNoticeVersion covers the shared identifier character rule for notice versions.
func TestParseNoticeVersion(t *testing.T) {
	// idCase is one row of the identifier rule table.
	type idCase struct {
		name string
		raw  string
		ok   bool
	}
	cases := []idCase{
		{"example shape", "notice-261102", true},
		{"single char", "a", true},
		{"allowed punctuation", "v1.2_3:4-5", true},
		{"max length", strings.Repeat("a", 64), true},
		{"empty", "", false},
		{"too long", strings.Repeat("a", 65), false},
		{"leading space", " notice", false},
		{"trailing space", "notice ", false},
		{"leading dash", "-notice", false},
		{"inner space", "no tice", false},
		{"non ascii", "notisé", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := candidateinterview.ParseNoticeVersion(tc.raw)
			if !tc.ok {
				require.ErrorIs(t, err, candidateinterview.ErrInvalidNoticeVersion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.raw, v.String())
		})
	}
}

// TestParseControllerID covers the same character rule under the distinct controller error.
func TestParseControllerID(t *testing.T) {
	c, err := candidateinterview.ParseControllerID("tab-1")
	require.NoError(t, err)
	assert.Equal(t, "tab-1", c.String())
	_, err = candidateinterview.ParseControllerID(" tab")
	require.ErrorIs(t, err, candidateinterview.ErrInvalidControllerID)
	_, err = candidateinterview.ParseControllerID("")
	require.ErrorIs(t, err, candidateinterview.ErrInvalidControllerID)
	_, err = candidateinterview.ParseControllerID(strings.Repeat("a", 65))
	require.ErrorIs(t, err, candidateinterview.ErrInvalidControllerID)
}

// TestNewConsent covers acceptance of the current version and refusal of every other input.
func TestNewConsent(t *testing.T) {
	current := mustVersion(t, "notice-261102")
	c, err := candidateinterview.NewConsent("c-1", "a-1", "s-1", current, current, fixtureNow())
	require.NoError(t, err)
	snap := c.Snapshot()
	assert.Equal(t, "c-1", snap.ID)
	assert.Equal(t, "a-1", snap.AssignmentID)
	assert.Equal(t, "s-1", snap.CandidateSessionID)
	assert.Equal(t, current, snap.NoticeVersion)
	assert.True(t, fixtureNow().Equal(snap.AcceptedAt))

	t.Run("other version refused", func(t *testing.T) {
		other := mustVersion(t, "notice-000000")
		_, err := candidateinterview.NewConsent("c-1", "a-1", "s-1", other, current, fixtureNow())
		require.ErrorIs(t, err, candidateinterview.ErrNoticeNotCurrent)
	})
	t.Run("zero version refused", func(t *testing.T) {
		_, err := candidateinterview.NewConsent("c-1", "a-1", "s-1", candidateinterview.NoticeVersion{}, current, fixtureNow())
		require.ErrorIs(t, err, candidateinterview.ErrInvalidConsent)
	})
	t.Run("empty ids and zero time refused", func(t *testing.T) {
		for _, args := range [][3]string{{"", "a", "s"}, {"c", "", "s"}, {"c", "a", ""}} {
			_, err := candidateinterview.NewConsent(args[0], args[1], args[2], current, current, fixtureNow())
			require.ErrorIs(t, err, candidateinterview.ErrInvalidConsent)
		}
		_, err := candidateinterview.NewConsent("c", "a", "s", current, current, time.Time{})
		require.ErrorIs(t, err, candidateinterview.ErrInvalidConsent)
	})
}

// TestRestoreConsent covers restoring a retired version and the stored acceptance time.
func TestRestoreConsent(t *testing.T) {
	retired := mustVersion(t, "notice-000000")
	at := fixtureNow()
	stored := candidateinterview.ConsentSnapshot{ID: "c-1", AssignmentID: "a-1", NoticeVersion: retired, CandidateSessionID: "s-1", AcceptedAt: at}
	c, err := candidateinterview.RestoreConsent(stored)
	require.NoError(t, err)
	assert.Equal(t, stored, c.Snapshot())
	assert.True(t, at.Equal(c.Snapshot().AcceptedAt))

	stored.ID = ""
	_, err = candidateinterview.RestoreConsent(stored)
	require.ErrorIs(t, err, candidateinterview.ErrInvalidConsent)
	stored.ID = "c-1"
	stored.AcceptedAt = time.Time{}
	_, err = candidateinterview.RestoreConsent(stored)
	require.ErrorIs(t, err, candidateinterview.ErrInvalidConsent)
}

// TestSessionCovers covers the exclusive coverage boundary and the budget constant.
func TestSessionCovers(t *testing.T) {
	assert.Equal(t, 15*time.Minute, candidateinterview.InterviewBudget)
	deadline := fixtureNow().Add(candidateinterview.InterviewBudget)
	assert.False(t, candidateinterview.SessionCovers(deadline, deadline), "equality is not covered")
	assert.True(t, candidateinterview.SessionCovers(deadline.Add(time.Millisecond), deadline))
	assert.False(t, candidateinterview.SessionCovers(deadline.Add(-time.Millisecond), deadline))
}

// TestClaimController covers the first claim at version 1.
func TestClaimController(t *testing.T) {
	id := mustController(t, "tab-1")
	c, err := candidateinterview.ClaimController("a-1", "e-1", id, fixtureNow())
	require.NoError(t, err)
	snap := c.Snapshot()
	assert.Equal(t, int64(1), snap.Version)
	assert.Equal(t, id, snap.ControllerID)
	assert.Equal(t, "a-1", snap.AssignmentID)
	assert.Equal(t, "e-1", snap.ExecutionID)
	assert.True(t, c.IsCurrent(id))
	assert.False(t, c.IsCurrent(mustController(t, "tab-2")))

	_, err = candidateinterview.ClaimController("", "e-1", id, fixtureNow())
	require.ErrorIs(t, err, candidateinterview.ErrInvalidController)
	_, err = candidateinterview.ClaimController("a-1", "e-1", candidateinterview.ControllerID{}, fixtureNow())
	require.ErrorIs(t, err, candidateinterview.ErrInvalidController)
}

// TestInterviewControllerTakeover covers increment, stale refusal, and terminal refusal.
func TestInterviewControllerTakeover(t *testing.T) {
	first, second := mustController(t, "tab-1"), mustController(t, "tab-2")
	later := fixtureNow().Add(time.Second)

	t.Run("increments version and replaces id", func(t *testing.T) {
		c, err := candidateinterview.ClaimController("a-1", "e-1", first, fixtureNow())
		require.NoError(t, err)
		require.NoError(t, c.Takeover(1, second, false, later))
		snap := c.Snapshot()
		assert.Equal(t, int64(2), snap.Version)
		assert.Equal(t, second, snap.ControllerID)
		assert.True(t, later.Equal(snap.UpdatedAt))
		assert.False(t, c.IsCurrent(first))
	})
	t.Run("stale version keeps previous id", func(t *testing.T) {
		c, err := candidateinterview.ClaimController("a-1", "e-1", first, fixtureNow())
		require.NoError(t, err)
		require.ErrorIs(t, c.Takeover(7, second, false, later), candidateinterview.ErrStaleControllerVersion)
		snap := c.Snapshot()
		assert.Equal(t, int64(1), snap.Version)
		assert.Equal(t, first, snap.ControllerID)
		assert.True(t, fixtureNow().Equal(snap.UpdatedAt))
	})
	t.Run("terminal execution refuses", func(t *testing.T) {
		c, err := candidateinterview.ClaimController("a-1", "e-1", first, fixtureNow())
		require.NoError(t, err)
		require.ErrorIs(t, c.Takeover(1, second, true, later), candidateinterview.ErrExecutionTerminal)
		assert.Equal(t, first, c.Snapshot().ControllerID)
	})
	t.Run("zero id refused", func(t *testing.T) {
		c, err := candidateinterview.ClaimController("a-1", "e-1", first, fixtureNow())
		require.NoError(t, err)
		require.ErrorIs(t, c.Takeover(1, candidateinterview.ControllerID{}, false, later), candidateinterview.ErrInvalidControllerID)
	})
}

// TestRestoreController covers stored shape checks and snapshot detachment.
func TestRestoreController(t *testing.T) {
	stored := candidateinterview.ControllerSnapshot{AssignmentID: "a-1", ExecutionID: "e-1", ControllerID: mustController(t, "tab-1"), Version: 3, UpdatedAt: fixtureNow()}
	c, err := candidateinterview.RestoreController(stored)
	require.NoError(t, err)
	assert.Equal(t, stored, c.Snapshot())

	stored.Version = 0
	_, err = candidateinterview.RestoreController(stored)
	require.ErrorIs(t, err, candidateinterview.ErrInvalidController)
}
