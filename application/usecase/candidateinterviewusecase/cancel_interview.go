package candidateinterviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// CancelInterview asks lifecycle to commit a candidate early stop for the current controller.
type CancelInterview struct {
	Current     service.CurrentCandidateSession
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
}

// Execute cancels the interview when the caller is the current controller.
func (u CancelInterview) Execute(_ context.Context, _ CancelInterviewInput) (*dto.InterviewExecutionView, error) {
	return nil, errNotImplemented
}

var _ CancelInterviewOperation = CancelInterview{}
