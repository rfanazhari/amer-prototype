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
	if !s.Identity.valid() || s.ValidationID == "" || s.CreatedAt.IsZero() {
		return PublishedRevision[C]{}, ErrInvalid
	}
	s.Configuration = s.Configuration.Clone()
	return PublishedRevision[C]{state: s}, nil
}

// Snapshot returns a deep copy of the immutable revision.
func (r PublishedRevision[C]) Snapshot() RevisionSnapshot[C] {
	s := r.state
	s.Configuration = s.Configuration.Clone()
	return s
}

// valid checks owned revision identity independently of current template policy.
func (r RevisionIdentity) valid() bool {
	return r.ID != "" && r.InstanceID != "" && r.Number > 0 && r.DraftVersion.number > 0 && r.Template.valid()
}
