package service

import "errors"

var (
	// ErrReportInvalidInput reports an interview report command with a missing or malformed identity.
	ErrReportInvalidInput = errors.New("interview report input is invalid")
	// ErrReportNotCompleted reports that the interview is not yet completed, so no report can be admitted.
	ErrReportNotCompleted = errors.New("interview report is available only after the interview is completed")
	// ErrReportEvidenceUnavailable reports that the evidence port lists no finalized record for the interview.
	ErrReportEvidenceUnavailable = errors.New("interview report needs the finalized interview record")
	// ErrReportIdempotencyConflict reports a request key already used with a different regeneration intent.
	ErrReportIdempotencyConflict = errors.New("interview report request identity is already used with a different intent")
)
