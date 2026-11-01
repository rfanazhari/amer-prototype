package candidateinterviewusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
)

// CancelInterview asks lifecycle to commit a candidate early stop for the current controller.
type CancelInterview struct {
	Current     service.CurrentCandidateSession
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
}

// Execute cancels the interview. A foreign execution id is not found, a terminal execution is refused so it is never
// relabelled, and a tab that is not the current controller is refused before the lifecycle cancel call.
func (u CancelInterview) Execute(ctx context.Context, in CancelInterviewInput) (*dto.InterviewExecutionView, error) {
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
	if !live(facts.State) {
		return nil, candidateinterview.ErrExecutionTerminal
	}
	controller, err := u.Controllers.Find(ctx, session.AssignmentID)
	if errors.Is(err, candidateinterviewport.ErrNotFound) || (err == nil && controller.ControllerID != tab) {
		return nil, ErrNotController
	}
	if err != nil {
		return nil, err
	}
	cancelled, err := u.Lifecycle.Cancel(ctx, service.CancelRequest{
		ExecutionID: in.ExecutionID, ExpectedVersion: in.ExpectedVersion, Reason: service.CancelReasonCandidateEarlyStop,
	})
	if err != nil {
		return nil, err
	}
	view := executionView(cancelled, true, controller.Version)
	return &view, nil
}

var _ CancelInterviewOperation = CancelInterview{}
