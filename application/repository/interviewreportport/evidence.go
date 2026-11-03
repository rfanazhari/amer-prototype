package interviewreportport

import (
	"context"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

// EvidenceEntry identifies one finalized interview record that a statement may cite.
type EvidenceEntry struct {
	// ID is the stable finalized entry identity.
	ID string
	// Revision is the positive revision of the entry.
	Revision int
	// QuestionID is the coverage identity of the entry, empty when the record supplies none.
	QuestionID string
}

// Evidence reads finalized interview records. It has no production adapter yet.
type Evidence interface {
	// Finalized returns the finalized entries of one execution, or an empty slice when none are final.
	// Partial transcript fragments are never returned.
	Finalized(ctx context.Context, executionID string) ([]EvidenceEntry, error)
}

// WriteRequest is the input of one report writing attempt.
// It carries no salary ceiling value or reference.
type WriteRequest struct {
	// ExecutionID is the interview execution being described.
	ExecutionID string
	// Evidence lists the finalized entries the writer may cite.
	Evidence []EvidenceEntry
}

// Writer produces the structured account for one attempt. It has no production adapter yet.
type Writer interface {
	// Write returns candidate statements. The caller validates them as writer output
	// before storing any, so a rejected statement fails the attempt and not the interview.
	Write(ctx context.Context, request WriteRequest) ([]interviewreport.StatementInput, error)
}
