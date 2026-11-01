package candidateinterviewquery

import (
	"context"

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

// Execute returns the candidate-safe execution through the lifecycle read only, never commit-or-find. An execution
// that is not the assignment's one is not found; an unbound lifecycle adapter is a dependency failure.
func (q GetExecution) Execute(ctx context.Context, in GetExecutionInput) (*dto.InterviewExecutionView, error) {
	session, err := q.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	facts, err := q.Lifecycle.Find(ctx, session.AssignmentID)
	if err != nil {
		return nil, err
	}
	if facts.ID != in.ExecutionID {
		return nil, service.ErrInterviewNotFound
	}
	view, err := safeExecution(ctx, q.Controllers, facts, session.AssignmentID, in.ControllerID)
	if err != nil {
		return nil, err
	}
	return view, nil
}

var _ GetExecutionOperation = GetExecution{}
