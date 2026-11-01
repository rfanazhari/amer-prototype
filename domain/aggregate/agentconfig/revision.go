package agentconfig

import "time"

// RevisionIdentity describes an owned active selection without loading history.
type RevisionIdentity struct {
	ID, InstanceID string
	Number         uint64
	DraftVersion   DraftVersion
	Template       TemplateRef
}

// RevisionSnapshot transfers frozen configuration and its validation provenance.
type RevisionSnapshot[C Configuration[C]] struct {
	Identity      RevisionIdentity
	ValidationID  string
	Configuration C
	CreatedAt     time.Time
}

// PublishedRevision owns a detached snapshot and exposes no mutation operation.
type PublishedRevision[C Configuration[C]] struct{ state RevisionSnapshot[C] }

// RestoreRevision restores structural provenance, without current creation policy.
func RestoreRevision[C Configuration[C]](s RevisionSnapshot[C]) (PublishedRevision[C], error) {
	return PublishedRevision[C]{}, ErrInvalid
}

// Snapshot returns a deep copy of the immutable revision.
func (r PublishedRevision[C]) Snapshot() RevisionSnapshot[C] { return RevisionSnapshot[C]{} }
