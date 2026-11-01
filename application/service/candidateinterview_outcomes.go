package service

import "errors"

var (
	// ErrInterviewNotLive reports a lifecycle refusal because the execution is not live.
	ErrInterviewNotLive = errors.New("interview is not live")
	// ErrInterviewStartUncertain reports an ambiguous commit that must be retried with the same intent.
	ErrInterviewStartUncertain = errors.New("interview start outcome is uncertain")
)
