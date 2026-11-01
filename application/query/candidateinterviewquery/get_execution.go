package candidateinterviewquery

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// GetExecution reads one execution inside the session's assignment.
type GetExecution struct {
	Current     service.CurrentCandidateSession
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
}

// Execute returns the candidate-safe execution, or a not-found error outside the assignment.
func (q GetExecution) Execute(_ context.Context, _ GetExecutionInput) (*dto.InterviewExecutionView, error) {
	return nil, errors.New("not implemented")
}

var _ GetExecutionOperation = GetExecution{}
