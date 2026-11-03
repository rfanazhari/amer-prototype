package service

import "errors"

var (
	// ErrReportInvalidInput reports an interview report command with a missing or malformed identity.
	ErrReportInvalidInput = errors.New("interview report input is invalid")
	// ErrReportNotCompleted reports that the interview is not yet completed, so no report can be admitted.
	ErrReportNotCompleted = errors.New("interview report is available only after the interview is completed")
)
