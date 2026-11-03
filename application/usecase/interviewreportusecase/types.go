package interviewreportusecase

// AdmitInput identifies the completed interview that is admitted for reporting.
type AdmitInput struct {
	// AssignmentID is the interview assignment whose completion marker was set by the caller.
	AssignmentID string
	// ExecutionID is the interview execution pinned to that assignment.
	ExecutionID string
}
