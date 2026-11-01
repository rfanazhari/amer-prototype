package agentconfig_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfiguration is a minimal editable value, not a production template schema.
type testConfiguration struct {
	Values  []string
	Expires *time.Time
}

// Clone implements value ownership, including nested slices and optional times.
func (c testConfiguration) Clone() testConfiguration {
	c.Values = append([]string(nil), c.Values...)
	if c.Expires != nil {
		v := *c.Expires
		c.Expires = &v
	}
	return c
}

// fixtureTime supplies reproducible trusted UTC millisecond time.
func fixtureTime() time.Time { return time.Date(2026, 11, 1, 1, 0, 0, 123000000, time.UTC) }

// fixtureVersion constructs an explicitly chosen initial sequence, without creation policy.
func fixtureVersion(t *testing.T, n uint64) agentconfig.DraftVersion {
	t.Helper()
	v, err := agentconfig.NewDraftVersion(n)
	require.NoError(t, err)
	return v
}

// fixtureState supplies an incomplete draft with mutable values to test ownership.
func fixtureState(t *testing.T) agentconfig.InstanceSnapshot[testConfiguration] {
	t.Helper()
	expires := fixtureTime().Add(time.Hour)
	return agentconfig.InstanceSnapshot[testConfiguration]{ID: "instance-a", Template: agentconfig.TemplateRef{Template: "fixture", Version: "v1", Behavior: "b1"}, DraftVersion: fixtureVersion(t, 4), Configuration: testConfiguration{Values: []string{"original"}, Expires: &expires}, CreatedAt: fixtureTime(), UpdatedAt: fixtureTime()}
}

// fixtureInstance creates an unpublished aggregate through its public constructor.
func fixtureInstance(t *testing.T) *agentconfig.AgentInstance[testConfiguration] {
	t.Helper()
	a, err := agentconfig.NewInstance(fixtureState(t))
	require.NoError(t, err)
	return a
}

// fixtureValidation creates a detached exact-version assessment for an instance.
func fixtureValidation(t *testing.T, s agentconfig.InstanceSnapshot[testConfiguration], eligible bool) agentconfig.Validation {
	t.Helper()
	v, err := agentconfig.NewValidation(agentconfig.ValidationSnapshot{ID: "validation", InstanceID: s.ID, DraftVersion: s.DraftVersion, Template: s.Template, Eligible: eligible, Findings: []agentconfig.Finding{{Code: "fixture", Path: "values"}}, CreatedAt: s.UpdatedAt})
	require.NoError(t, err)
	return v
}

// TestDraftVersion proves positive identity and zero-value rejection.
func TestDraftVersion(t *testing.T) {
	// versionCase describes accepted sequence values.
	type versionCase struct {
		name    string
		number  uint64
		wantErr bool
	}
	for _, tt := range []versionCase{{"zero", 0, true}, {"one", 1, false}, {"historical", 42, false}, {"maximum", ^uint64(0), false}} {
		t.Run(tt.name, func(t *testing.T) {
			v, err := agentconfig.NewDraftVersion(tt.number)
			if tt.wantErr {
				require.ErrorIs(t, err, agentconfig.ErrInvalid)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.number, v.Number())
		})
	}
}

// TestInstanceReplacement proves version comparison, failure atomicity and active protection.
func TestInstanceReplacement(t *testing.T) {
	// replacementCase selects valid, stale, zero and exhausted replacement sequences.
	type replacementCase struct {
		name              string
		current, expected uint64
		wantErr           bool
		target            error
	}
	for _, tt := range []replacementCase{{"replace", 4, 4, false, nil}, {"stale", 4, 3, true, agentconfig.ErrStale}, {"future", 4, 5, true, agentconfig.ErrStale}, {"zero", 4, 0, true, agentconfig.ErrStale}, {"overflow", ^uint64(0), ^uint64(0), true, agentconfig.ErrInvalid}} {
		t.Run(tt.name, func(t *testing.T) {
			s := fixtureState(t)
			s.DraftVersion = fixtureVersion(t, tt.current)
			a, err := agentconfig.NewInstance(s)
			require.NoError(t, err)
			_, err = a.Publish("revision-1", fixtureValidation(t, a.Snapshot(), true), fixtureTime())
			require.NoError(t, err)
			before := a.Snapshot()
			expected := agentconfig.DraftVersion{}
			if tt.expected != 0 {
				expected = fixtureVersion(t, tt.expected)
			}
			input := testConfiguration{Values: []string{"edited"}}
			err = a.ReplaceDraft(expected, input, fixtureTime().Add(time.Millisecond))
			if tt.wantErr {
				require.ErrorIs(t, err, tt.target)
				assert.Equal(t, before, a.Snapshot())
				return
			}
			require.NoError(t, err)
			input.Values[0] = "caller mutation"
			after := a.Snapshot()
			assert.Equal(t, uint64(5), after.DraftVersion.Number())
			assert.Equal(t, []string{"edited"}, after.Configuration.Values)
			assert.Nil(t, after.Configuration.Expires)
			assert.Equal(t, before.Active, after.Active)
			assert.Equal(t, before.CreatedAt, after.CreatedAt)
			assert.Equal(t, fixtureTime().Add(time.Millisecond), after.UpdatedAt)
		})
	}
}

// TestInstanceValidationAndPublication proves exact ownership and eligibility guards.
func TestInstanceValidationAndPublication(t *testing.T) {
	// validationCase mutates one independent assessment fact.
	type validationCase struct {
		name     string
		change   func(*agentconfig.ValidationSnapshot)
		wantErr  bool
		checkErr error
	}
	for _, tt := range []validationCase{
		{"matching", func(*agentconfig.ValidationSnapshot) {}, false, nil},
		{"foreign", func(s *agentconfig.ValidationSnapshot) { s.InstanceID = "other" }, true, agentconfig.ErrValidation},
		{"stale", func(s *agentconfig.ValidationSnapshot) { s.DraftVersion = fixtureVersion(t, 3) }, true, agentconfig.ErrValidation},
		{"template", func(s *agentconfig.ValidationSnapshot) { s.Template.Template = "other" }, true, agentconfig.ErrValidation},
		{"template version", func(s *agentconfig.ValidationSnapshot) { s.Template.Version = "v2" }, true, agentconfig.ErrValidation},
		{"behavior", func(s *agentconfig.ValidationSnapshot) { s.Template.Behavior = "b2" }, true, agentconfig.ErrValidation},
		{"ineligible", func(s *agentconfig.ValidationSnapshot) { s.Eligible = false }, true, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := fixtureInstance(t)
			s := fixtureValidation(t, a.Snapshot(), true).Snapshot()
			tt.change(&s)
			v, err := agentconfig.NewValidation(s)
			require.NoError(t, err)
			assert.ErrorIs(t, a.CheckValidation(v), tt.checkErr)
			before := a.Snapshot()
			revision, err := a.Publish("revision-1", v, fixtureTime())
			if tt.wantErr {
				require.ErrorIs(t, err, agentconfig.ErrValidation)
				assert.Equal(t, before, a.Snapshot())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, uint64(1), revision.Snapshot().Identity.Number)
			assert.Equal(t, "revision-1", a.Snapshot().Active.ID)
			assert.Equal(t, "validation", revision.Snapshot().ValidationID)
			_, err = a.Publish("revision-1", v, fixtureTime())
			require.ErrorIs(t, err, agentconfig.ErrInvalid)
		})
	}
}

// TestInstanceSnapshots proves copying at construction, observation and publication boundaries.
func TestInstanceSnapshots(t *testing.T) {
	input := fixtureState(t)
	a, err := agentconfig.NewInstance(input)
	require.NoError(t, err)
	input.Configuration.Values[0] = "input mutation"
	*input.Configuration.Expires = time.Time{}
	before := a.Snapshot()
	assert.Equal(t, []string{"original"}, before.Configuration.Values)
	assert.Equal(t, fixtureTime().Add(time.Hour), *before.Configuration.Expires)
	v := fixtureValidation(t, before, true)
	r, err := a.Publish("revision-1", v, fixtureTime())
	require.NoError(t, err)
	exposed := a.Snapshot()
	exposed.Configuration.Values[0] = "output mutation"
	*exposed.Configuration.Expires = time.Time{}
	exposed.Active.ID = "wrong"
	frozen := r.Snapshot()
	frozen.Configuration.Values[0] = "revision mutation"
	*frozen.Configuration.Expires = time.Time{}
	assert.Equal(t, []string{"original"}, a.Snapshot().Configuration.Values)
	assert.Equal(t, "revision-1", a.Snapshot().Active.ID)
	err = a.ReplaceDraft(before.DraftVersion, testConfiguration{}, fixtureTime().Add(time.Second))
	require.NoError(t, err)
	assert.ErrorIs(t, a.CheckValidation(v), agentconfig.ErrValidation)
	r2, err := a.Publish("revision-2", fixtureValidation(t, a.Snapshot(), true), fixtureTime().Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, uint64(2), r2.Snapshot().Identity.Number)
	assert.Equal(t, []string{"original"}, r.Snapshot().Configuration.Values)
	assert.Equal(t, fixtureTime().Add(time.Hour), *r.Snapshot().Configuration.Expires)
	assert.Equal(t, uint64(4), r.Snapshot().Identity.DraftVersion.Number())
}

// TestInstanceRestoration distinguishes history loading from new-instance creation.
func TestInstanceRestoration(t *testing.T) {
	a := fixtureInstance(t)
	r, err := a.Publish("revision", fixtureValidation(t, a.Snapshot(), true), fixtureTime())
	require.NoError(t, err)
	// restoreCase describes structural corruption independent of template field policy.
	type restoreCase struct {
		name    string
		change  func(*agentconfig.InstanceSnapshot[testConfiguration])
		wantErr bool
	}
	for _, tt := range []restoreCase{
		{"historical", func(*agentconfig.InstanceSnapshot[testConfiguration]) {}, false},
		{"identity", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.ID = "" }, true},
		{"version", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.DraftVersion = agentconfig.DraftVersion{} }, true},
		{"template", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.Template.Version = "" }, true},
		{"behavior", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.Template.Behavior = "" }, true},
		{"time", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.UpdatedAt = s.CreatedAt.Add(-time.Second) }, true},
		{"foreign active", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.Active.InstanceID = "other" }, true},
		{"active template", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.Active.Template.Version = "other" }, true},
		{"active future", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.Active.DraftVersion = fixtureVersion(t, 5) }, true},
		{"active zero", func(s *agentconfig.InstanceSnapshot[testConfiguration]) { s.Active.Number = 0 }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := a.Snapshot()
			tt.change(&s)
			restored, err := agentconfig.RestoreInstance(s)
			if tt.wantErr {
				require.ErrorIs(t, err, agentconfig.ErrInvalid)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, s, restored.Snapshot())
			_, err = agentconfig.NewInstance(s)
			require.ErrorIs(t, err, agentconfig.ErrInvalid)
			s.Configuration.Values[0] = "changed"
			s.Active.ID = "changed"
			assert.Equal(t, "revision", restored.Snapshot().Active.ID)
			assert.Equal(t, []string{"original"}, restored.Snapshot().Configuration.Values)
		})
	}
	rs := r.Snapshot()
	restored, err := agentconfig.RestoreRevision(rs)
	require.NoError(t, err)
	rs.Configuration.Values[0] = "changed"
	assert.Equal(t, []string{"original"}, restored.Snapshot().Configuration.Values)
}

// TestValidationRestoration proves detached findings and structural history checks.
func TestValidationRestoration(t *testing.T) {
	s := fixtureValidation(t, fixtureState(t), false).Snapshot()
	v, err := agentconfig.RestoreValidation(s)
	require.NoError(t, err)
	s.Findings[0].Code = "changed"
	out := v.Snapshot()
	assert.Equal(t, "fixture", out.Findings[0].Code)
	out.Findings[0].Code = "changed again"
	assert.Equal(t, "fixture", v.Snapshot().Findings[0].Code)
	// validationRestoreCase selects corrupt identity fields.
	type validationRestoreCase struct {
		name    string
		change  func(*agentconfig.ValidationSnapshot)
		wantErr bool
	}
	for _, tt := range []validationRestoreCase{
		{"missing id", func(s *agentconfig.ValidationSnapshot) { s.ID = "" }, true},
		{"missing owner", func(s *agentconfig.ValidationSnapshot) { s.InstanceID = "" }, true},
		{"zero draft", func(s *agentconfig.ValidationSnapshot) { s.DraftVersion = agentconfig.DraftVersion{} }, true},
		{"missing template", func(s *agentconfig.ValidationSnapshot) { s.Template.Template = "" }, true},
		{"missing time", func(s *agentconfig.ValidationSnapshot) { s.CreatedAt = time.Time{} }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := v.Snapshot()
			tt.change(&state)
			_, err := agentconfig.RestoreValidation(state)
			require.ErrorIs(t, err, agentconfig.ErrInvalid)
		})
	}
}

// TestRevisionRestoration rejects corrupt provenance without invoking field validators.
func TestRevisionRestoration(t *testing.T) {
	a := fixtureInstance(t)
	r, err := a.Publish("revision", fixtureValidation(t, a.Snapshot(), true), fixtureTime())
	require.NoError(t, err)
	// revisionCase selects persisted structural corruption.
	type revisionCase struct {
		name    string
		change  func(*agentconfig.RevisionSnapshot[testConfiguration])
		wantErr bool
	}
	for _, tt := range []revisionCase{
		{"id", func(s *agentconfig.RevisionSnapshot[testConfiguration]) { s.Identity.ID = "" }, true},
		{"owner", func(s *agentconfig.RevisionSnapshot[testConfiguration]) { s.Identity.InstanceID = "" }, true},
		{"number", func(s *agentconfig.RevisionSnapshot[testConfiguration]) { s.Identity.Number = 0 }, true},
		{"draft", func(s *agentconfig.RevisionSnapshot[testConfiguration]) {
			s.Identity.DraftVersion = agentconfig.DraftVersion{}
		}, true},
		{"template", func(s *agentconfig.RevisionSnapshot[testConfiguration]) { s.Identity.Template.Version = "" }, true},
		{"validation", func(s *agentconfig.RevisionSnapshot[testConfiguration]) { s.ValidationID = "" }, true},
		{"time", func(s *agentconfig.RevisionSnapshot[testConfiguration]) { s.CreatedAt = time.Time{} }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := r.Snapshot()
			tt.change(&s)
			_, err := agentconfig.RestoreRevision(s)
			require.ErrorIs(t, err, agentconfig.ErrInvalid)
		})
	}
}

// TestInstanceZeroValue rejects mutation through an uninitialized aggregate or assessment.
func TestInstanceZeroValue(t *testing.T) {
	var a agentconfig.AgentInstance[testConfiguration]
	require.ErrorIs(t, a.ReplaceDraft(agentconfig.DraftVersion{}, testConfiguration{}, fixtureTime()), agentconfig.ErrInvalid)
	require.ErrorIs(t, a.CheckValidation(agentconfig.Validation{}), agentconfig.ErrInvalid)
	_, err := a.Publish("revision", agentconfig.Validation{}, fixtureTime())
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
	valid := fixtureInstance(t)
	require.ErrorIs(t, valid.CheckValidation(agentconfig.Validation{}), agentconfig.ErrValidation)
}
