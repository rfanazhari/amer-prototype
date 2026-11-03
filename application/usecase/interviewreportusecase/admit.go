package interviewreportusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Admit records the first report generation for an interview completed in the caller's transaction.
type Admit struct {
	Transactions interviewreportport.Transaction
	Completion   interviewreportport.CompletionMarker
	Reports      interviewreportport.ReportReader
	Admissions   interviewreportport.AdmissionWriter
	Clock        service.Clock
	// NewID mints identities in the order report, generation, job, outbox; nil uses random UUIDs.
	NewID func() string
}

// Execute admits the interview. It is an inert stub until the behavior is implemented.
func (u Admit) Execute(ctx context.Context, input AdmitInput) (*dto.ReportGeneration, error) {
	return nil, nil
}
