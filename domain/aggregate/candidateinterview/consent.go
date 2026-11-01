package candidateinterview

import "time"

// ConsentSnapshot is the stored shape of one acceptance.
type ConsentSnapshot struct {
	// ID is the consent identity.
	ID string
	// AssignmentID is the assignment that accepted.
	AssignmentID string
	// NoticeVersion is the accepted notice version.
	NoticeVersion NoticeVersion
	// CandidateSessionID is the session that carried the acceptance.
	CandidateSessionID string
	// AcceptedAt is the trusted acceptance instant.
	AcceptedAt time.Time
}

// Consent is the aggregate root for one acceptance of one notice version.
type Consent struct{ state ConsentSnapshot }

// NewConsent records an acceptance of the current notice version.
func NewConsent(id, assignmentID, sessionID string, accepted, current NoticeVersion, acceptedAt time.Time) (*Consent, error) {
	toStore := ConsentSnapshot{ID: id, AssignmentID: assignmentID, NoticeVersion: accepted, CandidateSessionID: sessionID, AcceptedAt: acceptedAt}
	if !toStore.consistent() {
		return nil, ErrInvalidConsent
	}
	if accepted != current {
		return nil, ErrNoticeNotCurrent
	}
	return &Consent{state: toStore}, nil
}

// consistent reports whether every field of the stored acceptance is present.
func (s ConsentSnapshot) consistent() bool {
	return s.ID != "" && s.AssignmentID != "" && s.CandidateSessionID != "" &&
		s.NoticeVersion.value != "" && !s.AcceptedAt.IsZero()
}

// RestoreConsent rebuilds a consent from stored data, whatever the current notice version is.
func RestoreConsent(s ConsentSnapshot) (*Consent, error) {
	if !s.consistent() {
		return nil, ErrInvalidConsent
	}
	return &Consent{state: s}, nil
}

// Snapshot returns a detached copy of the consent.
func (c *Consent) Snapshot() ConsentSnapshot { return c.state }
