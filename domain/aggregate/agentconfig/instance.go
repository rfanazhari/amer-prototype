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
	return nil, ErrInvalid
}

// RestoreInstance restores current facts, including an owned active selection.
func RestoreInstance[C Configuration[C]](s InstanceSnapshot[C]) (*AgentInstance[C], error) {
	return nil, ErrInvalid
}

// Snapshot returns detached current facts and optional active selection.
func (a *AgentInstance[C]) Snapshot() InstanceSnapshot[C] { return InstanceSnapshot[C]{} }

// ReplaceDraft accepts only the expected current version and preserves active selection.
func (a *AgentInstance[C]) ReplaceDraft(expected DraftVersion, configuration C, now time.Time) error {
	return ErrInvalid
}

// CheckValidation rejects foreign, stale or differently interpreted assessments.
// Eligibility is checked only by publication, allowing invalid drafts to be assessed.
func (a *AgentInstance[C]) CheckValidation(v Validation) error { return ErrInvalid }

// Publish freezes the exact eligible draft and advances active selection once.
// Application replay must precede this operation inside the publication transaction.
func (a *AgentInstance[C]) Publish(id string, v Validation, now time.Time) (PublishedRevision[C], error) {
	return PublishedRevision[C]{}, ErrInvalid
}
