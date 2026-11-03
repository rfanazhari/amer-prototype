package candidateinterviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// CompleteInterview lets the controlling tab end a running interview as COMPLETED.
type CompleteInterview struct {
	Current     service.CurrentCandidateSession
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
	Completer   service.InterviewCompleter
}

// Execute completes the interview for the current controller.
func (u CompleteInterview) Execute(ctx context.Context, in CompleteInterviewInput) (*dto.InterviewExecutionView, error) {
	return nil, service.ErrInterviewDependencyUnavailable
}

var _ CompleteInterviewOperation = CompleteInterview{}
