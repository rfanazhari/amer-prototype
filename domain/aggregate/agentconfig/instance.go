package agentconfig

import "time"

// InstanceSnapshot transfers the current owned draft and optional active identity.
type InstanceSnapshot[C Configuration[C]] struct {
	ID                   string
	Template             TemplateRef
	DraftVersion         DraftVersion
	Configuration        C
	Active               *RevisionIdentity
	CreatedAt, UpdatedAt time.Time
}

// AgentInstance owns draft replacement and selection of newly published revisions.
// It does not retain revision history or implement command replay/persistence.
type AgentInstance[C Configuration[C]] struct{ state InstanceSnapshot[C] }

// NewInstance creates an unpublished draft with caller-selected initial version.
func NewInstance[C Configuration[C]](s InstanceSnapshot[C]) (*AgentInstance[C], error) {
	if s.Active != nil {
		return nil, ErrInvalid
	}
	return RestoreInstance(s)
}

// RestoreInstance restores current facts, including an owned active selection.
func RestoreInstance[C Configuration[C]](s InstanceSnapshot[C]) (*AgentInstance[C], error) {
	if !s.valid() {
		return nil, ErrInvalid
	}
	return &AgentInstance[C]{state: s.detached()}, nil
}

// Snapshot returns detached current facts and optional active selection.
func (a *AgentInstance[C]) Snapshot() InstanceSnapshot[C] { return a.state.detached() }

// ReplaceDraft accepts only the expected current version and preserves active selection.
func (a *AgentInstance[C]) ReplaceDraft(expected DraftVersion, configuration C, now time.Time) error {
	if a == nil || !a.state.valid() {
		return ErrInvalid
	}
	if expected != a.state.DraftVersion {
		return ErrStale
	}
	if a.state.DraftVersion.number == ^uint64(0) || now.IsZero() || now.Before(a.state.UpdatedAt) {
		return ErrInvalid
	}
	configuration = configuration.Clone()
	a.state.Configuration = configuration
	a.state.DraftVersion.number++
	a.state.UpdatedAt = now
	return nil
}

// CheckValidation rejects foreign, stale or differently interpreted assessments.
// Eligibility is checked only by publication, allowing invalid drafts to be assessed.
func (a *AgentInstance[C]) CheckValidation(v Validation) error {
	if a == nil || !a.state.valid() {
		return ErrInvalid
	}
	s := v.state
	if s.ID == "" || s.CreatedAt.IsZero() || s.InstanceID != a.state.ID || s.DraftVersion != a.state.DraftVersion || s.Template != a.state.Template {
		return ErrValidation
	}
	return nil
}

// Publish freezes the exact eligible draft and advances active selection once.
// Application replay must precede this operation inside the publication transaction.
func (a *AgentInstance[C]) Publish(id string, v Validation, now time.Time) (PublishedRevision[C], error) {
	if err := a.CheckValidation(v); err != nil {
		return PublishedRevision[C]{}, err
	}
	if !v.state.Eligible {
		return PublishedRevision[C]{}, ErrValidation
	}
	if id == "" || now.IsZero() || now.Before(a.state.UpdatedAt) || now.Before(v.state.CreatedAt) {
		return PublishedRevision[C]{}, ErrInvalid
	}
	number := uint64(1)
	if a.state.Active != nil {
		if a.state.Active.ID == id || a.state.Active.Number == ^uint64(0) {
			return PublishedRevision[C]{}, ErrInvalid
		}
		number = a.state.Active.Number + 1
	}
	identity := RevisionIdentity{ID: id, InstanceID: a.state.ID, Number: number, DraftVersion: a.state.DraftVersion, Template: a.state.Template}
	revision, err := RestoreRevision(RevisionSnapshot[C]{Identity: identity, ValidationID: v.state.ID, Configuration: a.state.Configuration, CreatedAt: now})
	if err != nil {
		return PublishedRevision[C]{}, err
	}
	a.state.Active = &identity
	a.state.UpdatedAt = now
	return revision, nil
}

// valid checks current stored relationships without requiring publishable content.
func (s InstanceSnapshot[C]) valid() bool {
	if s.ID == "" || !s.Template.valid() || s.DraftVersion.number == 0 || s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() || s.UpdatedAt.Before(s.CreatedAt) {
		return false
	}
	return s.Active == nil || (s.Active.valid() && s.Active.InstanceID == s.ID && s.Active.Template == s.Template && s.Active.DraftVersion.number <= s.DraftVersion.number)
}

// detached copies configuration and selection, keeping transfer objects independent.
func (s InstanceSnapshot[C]) detached() InstanceSnapshot[C] {
	s.Configuration = s.Configuration.Clone()
	if s.Active != nil {
		active := *s.Active
		s.Active = &active
	}
	return s
}
