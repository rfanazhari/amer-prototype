package service

import "context"

// InterviewCompleter ends an interview at the candidate's finish: it seals the remaining stages and completes the execution.
type InterviewCompleter interface {
	// CompleteInterview seals every remaining stage in order and completes the execution; repeating it resumes a partial run.
	CompleteInterview(ctx context.Context, executionID string, expectedVersion int64) error
}
