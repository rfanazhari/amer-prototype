package candidateinterviewusecase

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
)

// Takeover replaces the controlling tab of the assignment's live interview.
type Takeover struct {
	Current     service.CurrentCandidateSession
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
	Clock       service.Clock
}

// Execute makes the calling tab the controller. The execution must belong to the session's assignment, a stale
// expected version is a conflict that keeps the previous controller, and a terminal execution is refused.
func (u Takeover) Execute(ctx context.Context, in TakeoverInput) (*dto.TakeoverView, error) {
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
	now := u.Clock.Now().UTC().Truncate(time.Millisecond)
	var version int64
	err = u.Controllers.Update(ctx, session.AssignmentID, func(c *candidateinterview.InterviewController) error {
		if err := c.Takeover(in.ExpectedVersion, tab, !live(facts.State), now); err != nil {
			return err
		}
		version = c.Snapshot().Version
		return nil
	})
	switch {
	case errors.Is(err, candidateinterview.ErrStaleControllerVersion):
		return nil, service.ErrInterviewConflict
	case errors.Is(err, candidateinterviewport.ErrNotFound):
		return nil, service.ErrInterviewNotFound
	case err != nil:
		return nil, err
	}
	return &dto.TakeoverView{IsCurrentController: true, ControllerVersion: version}, nil
}

var _ TakeoverOperation = Takeover{}
