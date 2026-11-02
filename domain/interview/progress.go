// Package interview owns progress rules for a pinned HR interview, independent of execution lifecycle and providers.
package interview

import (
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

var (
	ErrInvalid  = errors.New("invalid interview progress")
	ErrRestore  = errors.New("corrupt stored interview progress")
	ErrConflict = errors.New("interview progress conflict")
	ErrEvidence = errors.New("final stage evidence required")
	ErrBudget   = errors.New("interview budget exhausted")
)

// Snapshot is a detached persistence boundary, never a mutable aggregate handle.
type Snapshot struct {
	ExecutionID, RevisionID     string
	Configuration               valueobject.HRConfiguration
	RunningAt, StageActivatedAt time.Time
	CurrentStage                enum.QuestionStage
	FollowUpLimit               int
	Coverage                    []CoverageItem
	Entries                     []TranscriptEntry
	FollowUps                   []string
	Losses                      []TechnicalLoss
	Handoffs                    []Handoff
	Reconnect                   ReconnectState
}

// Progress protects interview invariants behind public operations.
type Progress struct{ state Snapshot }

// trustedTime preserves the repository's UTC millisecond representation.
func trustedTime(at time.Time) time.Time { return at.UTC().Truncate(time.Millisecond) }

// stages returns the fixed interview order, including the optionally questionless closing.
func stages() []enum.QuestionStage {
	return []enum.QuestionStage{enum.StageBackground, enum.StageScreening, enum.StageCompensation, enum.StageClosing}
}

// stageIndex returns the position in the fixed order, or minus one for an unknown stage.
func stageIndex(stage enum.QuestionStage) int {
	for i, s := range stages() {
		if s == stage {
			return i
		}
	}
	return -1
}

// stageSeconds selects a stored duration without reapplying publication eligibility.
func stageSeconds(c valueobject.HRConfiguration, stage enum.QuestionStage) int {
	s := c.StageSeconds()
	if s == nil {
		return 0
	}
	switch stage {
	case enum.StageBackground:
		return s.Background()
	case enum.StageScreening:
		return s.Screening()
	case enum.StageCompensation:
		return s.Compensation()
	case enum.StageClosing:
		return s.Closing()
	}
	return 0
}

// open reports whether the current stage has not already been sealed.
func (p *Progress) open() bool {
	return p.state.ExecutionID != "" && p.state.CurrentStage.Valid() && len(p.state.Handoffs) == stageIndex(p.state.CurrentStage)
}

// activeAt rejects pre-activation instants and the inclusive outer deadline.
func (p *Progress) activeAt(now time.Time) bool {
	return p.open() && !now.IsZero() && !now.Before(p.state.StageActivatedAt) && !p.HardDeadlineReached(now)
}

// NewProgress builds pending questions from the pinned revision and freezes cap two.
func NewProgress(id string, revision agentconfig.PublishedRevision[valueobject.HRConfiguration], runningAt time.Time) (*Progress, error) {
	r := revision.Snapshot()
	if id == "" || r.Identity.ID == "" || runningAt.IsZero() {
		return nil, ErrInvalid
	}
	at := trustedTime(runningAt)
	s := Snapshot{ExecutionID: id, RevisionID: r.Identity.ID, Configuration: r.Configuration.Clone(), RunningAt: at, StageActivatedAt: at, CurrentStage: enum.StageBackground, FollowUpLimit: 2}
	for _, q := range r.Configuration.MandatoryQuestions() {
		s.Coverage = append(s.Coverage, CoverageItem{QuestionID: q.ID(), Stage: q.Stage(), State: Pending})
	}
	p, err := RestoreProgress(s)
	if err != nil {
		return nil, ErrInvalid
	}
	return p, nil
}

// Snapshot returns detached progress state, including nested handoff slices and time pointers.
func (p *Progress) Snapshot() Snapshot { return p.state.detach() }

// detach severs mutable aliases on both restoration and observation boundaries.
func (s Snapshot) detach() Snapshot {
	s.Configuration = s.Configuration.Clone()
	s.Coverage = append([]CoverageItem(nil), s.Coverage...)
	s.Entries = append([]TranscriptEntry(nil), s.Entries...)
	s.FollowUps = append([]string(nil), s.FollowUps...)
	s.Losses = append([]TechnicalLoss(nil), s.Losses...)
	s.Handoffs = append([]Handoff(nil), s.Handoffs...)
	for i := range s.Handoffs {
		h := &s.Handoffs[i]
		h.Coverage = append([]CoverageItem(nil), h.Coverage...)
		h.EntryIDs = append([]string(nil), h.EntryIDs...)
		h.Losses = append([]TechnicalLoss(nil), h.Losses...)
	}
	if s.Reconnect.DetectedAt != nil {
		at := *s.Reconnect.DetectedAt
		s.Reconnect.DetectedAt = &at
	}
	return s
}
