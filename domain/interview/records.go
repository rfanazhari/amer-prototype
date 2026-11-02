package interview

import (
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"time"
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
