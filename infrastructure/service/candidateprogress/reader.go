// Package candidateprogress projects interview runtime progress onto the candidate-safe summary.
package candidateprogress

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Reader implements the candidate-safe progress port over the runtime progress store.
type Reader struct {
	Store interviewport.ProgressStore
}

var _ service.InterviewProgress = Reader{}

// Current is not implemented yet.
func (Reader) Current(context.Context, string) (*service.CandidateProgress, error) {
	return nil, nil
}
