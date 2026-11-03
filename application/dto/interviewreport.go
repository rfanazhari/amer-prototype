package dto

import (
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

// ReportGeneration is a detached view of one report generation.
type ReportGeneration struct {
	// ID is the stable generation identity.
	ID string
	// Number is the one-based position of the generation within its report.
	Number int
	// State is pending, running, succeeded or failed.
	State string
	// FailureReason is set exactly when State is failed.
	FailureReason *string
	// CreatedAt is the UTC instant at which the generation was admitted.
	CreatedAt time.Time
}

// ReportObservation is a detached observation that cites finalized evidence.
type ReportObservation struct {
	// Body is the statement text.
	Body string
	// EvidenceEntryID is the cited finalized entry.
	EvidenceEntryID string
	// EvidenceRevision is the cited revision of that entry.
	EvidenceRevision int
}

// ReportGap is a detached topic the interview did not cover. It cites no evidence.
type ReportGap struct {
	// Body is the statement text.
	Body string
	// QuestionID is the coverage identity when the evidence source supplied one, otherwise nil.
	QuestionID *string
}

// ReportUncertainty is a detached unclear point, with a citation only when the writer supplied one.
type ReportUncertainty struct {
	// Body is the statement text.
	Body string
	// EvidenceEntryID is the cited finalized entry, nil when the statement cites none.
	EvidenceEntryID *string
	// EvidenceRevision is the cited revision, nil when the statement cites none.
	EvidenceRevision *int
}

// ReportAccount is the detached account of one succeeded generation.
// It has no recommendation, rank, score or salary field.
type ReportAccount struct {
	// GenerationID is the succeeded generation that holds the account.
	GenerationID string
	// Observations lists the statements presented as answers.
	Observations []ReportObservation
	// Gaps lists the topics without an answer.
	Gaps []ReportGap
	// Uncertainties lists the unclear points.
	Uncertainties []ReportUncertainty
}

// InterviewReportView is the detached HR view: the newest generation and the latest succeeded account.
type InterviewReportView struct {
	// AssignmentID is the interview assignment.
	AssignmentID string
	// ExecutionID is the interview execution pinned to the assignment.
	ExecutionID string
	// Generation is the newest generation.
	Generation ReportGeneration
	// Report is the latest succeeded account, nil when none has succeeded.
	Report *ReportAccount
}

// NewReportGeneration copies a domain generation into a detached view.
func NewReportGeneration(g interviewreport.Generation) ReportGeneration {
	view := ReportGeneration{ID: g.ID, Number: g.Number, State: string(g.State), CreatedAt: g.CreatedAt.UTC()}
	if g.State == interviewreport.StateFailed {
		reason := g.FailureReason
		view.FailureReason = &reason
	}
	return view
}
