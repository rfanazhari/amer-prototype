package interviewusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// Ready records browser readiness on the current attempt of the session's own execution.
type Ready struct {
	Current     service.CurrentCandidateSession
	Executions  service.ExecutionLifecycle
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.InterviewLifecycle
	Progress    interviewport.ProgressStore
	Clock       service.Clock
}

var _ ReadyOperation = Ready{}

// Execute fences the attempt, then stores the readiness instant on its open session under the lifecycle lock.
// It touches only the session row: coverage, stage and transcript stay as committed.
func (u Ready) Execute(ctx context.Context, in ReadyInput) (*ReadyResult, error) {
	if in.AttemptID == "" || in.ExecutionID == "" || in.ControllerID == "" || in.ExpectedVersion < 1 {
		return nil, ErrAttemptInvalid
	}
	session, facts, err := authorizeCandidate(ctx, u.Current, u.Executions, u.Controllers, in.ExecutionID, in.ControllerID)
	if err != nil {
		return nil, err
	}
	authority := service.InterviewAttempt{ExecutionID: in.ExecutionID, AttemptID: in.AttemptID, ExpectedVersion: in.ExpectedVersion}
	if err = u.Lifecycle.Fence(ctx, authority); err != nil {
		if errors.Is(err, interview.ErrConflict) {
			return nil, ErrAttemptNotReady
		}
		return nil, classify(err)
	}
	record, err := u.Progress.Find(ctx, in.ExecutionID)
	if err != nil {
		return nil, classify(err)
	}
	now := trusted(u.Clock)
	result := ReadyResult{AttemptID: in.AttemptID, ExecutionVersion: facts.Version}
	err = u.Lifecycle.WithProgress(ctx, authority, func(tx context.Context) error {
		if err := u.stillController(tx, session.AssignmentID, in.ControllerID); err != nil {
			return err
		}
		return u.Progress.Update(tx, interviewport.ProgressWrite{ExecutionID: in.ExecutionID, AttemptID: in.AttemptID, ExpectedVersion: record.Version}, func(r *interviewport.ProgressRecord) error {
			for i := range r.Sessions {
				s := &r.Sessions[i]
				if s.AttemptID != in.AttemptID || s.ClosedAt != nil {
					continue
				}
				if s.ReadinessAt == nil {
					at := now
					s.ReadinessAt = &at
				}
				result.SessionID, result.SessionCondition = s.ID, sessionCondition(*s)
				return nil
			}
			return ErrAttemptNotReady
		})
	})
	if err != nil {
		return nil, classify(err)
	}
	return &result, nil
}

// stillController re-reads the controller under the lifecycle lock so a takeover committed after the first check
// refuses the previous tab. A takeover that commits after this read is not excluded; that limit is documented, not fenced.
func (u Ready) stillController(ctx context.Context, assignmentID, controllerID string) error {
	tab, err := candidateinterview.ParseControllerID(controllerID)
	if err != nil {
		return ErrAttemptInvalid
	}
	held, err := u.Controllers.Find(ctx, assignmentID)
	switch {
	case errors.Is(err, candidateinterviewport.ErrNotFound), err == nil && held.ControllerID != tab:
		return ErrAttemptConflict
	}
	return err
}
