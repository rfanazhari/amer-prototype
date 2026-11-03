package candidateinterviewusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
)

// stateCompleted is the lifecycle state a finished interview reports.
const stateCompleted = "COMPLETED"

// CompleteInterview lets the controlling tab end a running interview as COMPLETED.
type CompleteInterview struct {
	Current     service.CurrentCandidateSession
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
	Completer   service.InterviewCompleter
}

// Execute completes the interview for the current controller. A foreign execution id is not found, a created
// interview has nothing to complete, a cancelled or failed one is never relabelled, and a tab that is not the
// current controller is refused before the completer runs. A finished interview returns its view again.
func (u CompleteInterview) Execute(ctx context.Context, in CompleteInterviewInput) (*dto.InterviewExecutionView, error) {
	tab, err := candidateinterview.ParseControllerID(in.ControllerID)
	if err != nil {
		return nil, err
	}
	session, err := u.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	facts, err := u.Lifecycle.Find(ctx, session.AssignmentID)
	if err != nil {
		return nil, err
	}
	if facts.ID != in.ExecutionID {
		return nil, service.ErrInterviewNotFound
	}
	switch facts.State {
	case stateCompleted, stateRunning:
	case stateCreated:
		return nil, service.ErrInterviewNotLive
	default:
		return nil, candidateinterview.ErrExecutionTerminal
	}
	controller, err := u.Controllers.Find(ctx, session.AssignmentID)
	if errors.Is(err, candidateinterviewport.ErrNotFound) || (err == nil && controller.ControllerID != tab) {
		return nil, ErrNotController
	}
	if err != nil {
		return nil, err
	}
	if facts.State == stateRunning {
		if err = u.Completer.CompleteInterview(ctx, in.ExecutionID, in.ExpectedVersion); err != nil {
			return nil, err
		}
		if facts, err = u.Lifecycle.Find(ctx, session.AssignmentID); err != nil {
			return nil, err
		}
		if facts.State != stateCompleted {
			return nil, service.ErrInterviewConflict
		}
	}
	view := executionView(facts, true, controller.Version)
	return &view, nil
}

var _ CompleteInterviewOperation = CompleteInterview{}
