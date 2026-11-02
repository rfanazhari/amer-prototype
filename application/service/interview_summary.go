package service

import (
	"context"

	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// InterviewSummarizer optionally narrates an already authorized structured handoff.
// Failure must never prevent committing the structured evidence.
type InterviewSummarizer interface {
	// Summarize returns optional narrative without deciding stage authorization.
	Summarize(ctx context.Context, handoff interview.Handoff) (string, error)
}
