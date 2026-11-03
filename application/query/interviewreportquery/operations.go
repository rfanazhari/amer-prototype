package interviewreportquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// ReadInput identifies the assignment whose report is read.
type ReadInput struct {
	// AssignmentID is the interview assignment.
	AssignmentID string
}

// ReadOperation is the HR report read contract consumed by delivery.
type ReadOperation interface {
	// Execute authorizes current HR before lookup and returns the newest generation with the latest succeeded account.
	Execute(context.Context, ReadInput) (*dto.InterviewReportView, error)
}
