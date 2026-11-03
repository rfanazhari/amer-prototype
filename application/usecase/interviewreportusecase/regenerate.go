package interviewreportusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// RegenerateCommandKind is the durable command kind of an explicit regeneration.
const RegenerateCommandKind = "regenerate-interview-report/v1"

// Regenerate starts a new report generation for a completed interview, or replays an earlier request.
type Regenerate struct {
	Authority    service.InterviewReportAuthority
	Transactions interviewreportport.Transaction
	Assignments  interviewreportport.AssignmentLocker
	Reports      interviewreportport.ReportReader
	Commands     interviewreportport.RegenerationStore
	Evidence     interviewreportport.Evidence
	Clock        service.Clock
	// NewID mints identities in the order report (only when the report is new), generation, job, outbox, command;
	// nil uses random UUIDs.
	NewID func() string
}

var _ RegenerateOperation = Regenerate{}

// errNotImplemented marks the behavior that the GREEN commit supplies.
var errNotImplemented = errors.New("not implemented")

// Execute authorizes current HR, checks replay before eligibility and withholds the result until commit.
func (u Regenerate) Execute(ctx context.Context, input RegenerateInput) (*RegenerateResult, error) {
	return nil, errNotImplemented
}
