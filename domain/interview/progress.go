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

// CoverageState describes recording completeness, without a competence verdict.
type CoverageState string

const (
	Pending  CoverageState = "pending"
	Asked    CoverageState = "asked"
	Answered CoverageState = "answered"
	Omitted  CoverageState = "omitted"
)

// OmissionReason identifies an approved nontechnical omission.
type OmissionReason string

const (
	CandidateSkip              OmissionReason = "candidate_skip"
	NoAnswerAfterClarification OmissionReason = "no_answer_after_clarification"
	SoftBudgetExpired          OmissionReason = "soft_budget_expired"
)

// EvidenceState is supplied by a caller that has already classified the observation.
type EvidenceState string

const (
	Fragment EvidenceState = "fragment"
	Final    EvidenceState = "final"
)

// Speaker identifies the author of a classified transcript entry.
type Speaker string

const (
	Candidate Speaker = "candidate"
	Model     Speaker = "model"
)

// CoverageItem transfers one question's state and its evidence references.
type CoverageItem struct {
	QuestionID                                           string
	Stage                                                enum.QuestionStage
	State                                                CoverageState
	OmissionReason                                       OmissionReason
	ClarificationEntryID, AnswerEntryID, OmissionEntryID string
}

// TranscriptEntry transfers classified speech; it contains no provider event name.
type TranscriptEntry struct {
	ID      string
	Stage   enum.QuestionStage
	Speaker Speaker
	State   EvidenceState
	Body    string
}

// TechnicalLoss records a recording gap independently of question coverage.
type TechnicalLoss struct {
	ID, QuestionID string
	Resolved       bool
}

// Handoff transfers the source record; an empty target closes the interview.
type Handoff struct {
	Stage, Target enum.QuestionStage
	Coverage      []CoverageItem
	EntryIDs      []string
	Losses        []TechnicalLoss
	CommittedAt   time.Time
}

// ReconnectState transfers the backend detection time and frozen reconnect counters.
type ReconnectState struct {
	DetectedAt         *time.Time
	Episodes, Attempts int
}

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

// NewProgress builds pending questions from the pinned revision and freezes cap two.
func NewProgress(id string, revision agentconfig.PublishedRevision[valueobject.HRConfiguration], runningAt time.Time) (*Progress, error) {
	return nil, ErrInvalid
}

// RestoreProgress restores stored policy without publication reassessment.
func RestoreProgress(s Snapshot) (*Progress, error) { return nil, ErrRestore }

// Snapshot returns detached progress state.
func (p *Progress) Snapshot() Snapshot { return Snapshot{} }

// Ask moves a pending question into asked coverage.
func (p *Progress) Ask(questionID string) error { return ErrConflict }

// RecordEntry accepts classified transcript evidence without interpreting provider events.
func (p *Progress) RecordEntry(entry TranscriptEntry) error { return ErrEvidence }

// Clarify records exactly one finalized model clarification for an asked question.
func (p *Progress) Clarify(questionID, entryID string) error { return ErrEvidence }

// Answer records a final candidate citation and resolves that question's technical gaps.
func (p *Progress) Answer(questionID, entryID string) error { return ErrEvidence }

// Omit records one of three approved reasons when its prerequisites hold.
func (p *Progress) Omit(questionID string, reason OmissionReason, entryID string, now time.Time) error {
	return ErrConflict
}

// RecordTechnicalLoss preserves open coverage and blocks a stage proposal.
func (p *Progress) RecordTechnicalLoss(id, questionID string) error { return ErrConflict }

// FollowUp consumes the stored stage cap only with a final candidate citation.
func (p *Progress) FollowUp(entryID string) error { return ErrEvidence }

// Authorize checks a fixed next-stage proposal without changing progress.
func (p *Progress) Authorize(target enum.QuestionStage, now time.Time) error { return ErrConflict }

// CommitHandoff seals the source's structured evidence after authorization.
func (p *Progress) CommitHandoff(target enum.QuestionStage, now time.Time) error { return ErrConflict }

// ActivateStage requires a committed source handoff before starting the target timer.
func (p *Progress) ActivateStage(target enum.QuestionStage, now time.Time) error { return ErrConflict }

// SoftBudgetReached includes equality at the published stage deadline.
func (p *Progress) SoftBudgetReached(now time.Time) bool { return false }

// HardDeadlineReached includes equality at fifteen minutes from running.
func (p *Progress) HardDeadlineReached(now time.Time) bool { return false }

// ReadyForCompletion requires all coverage, resolved gaps and four sealed handoffs.
func (p *Progress) ReadyForCompletion() bool { return false }

// BeginReconnect starts a backend-detected episode under the stored episode ceiling.
func (p *Progress) BeginReconnect(now time.Time) error { return ErrBudget }

// AttemptReconnect consumes one replacement attempt strictly within 120 seconds.
func (p *Progress) AttemptReconnect(now time.Time) error { return ErrBudget }

// FinishReconnect ends the episode while retaining its counters.
func (p *Progress) FinishReconnect() error { return ErrConflict }
