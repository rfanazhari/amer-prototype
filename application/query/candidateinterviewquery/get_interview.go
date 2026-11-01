package candidateinterviewquery

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
)

// Action tokens published in the candidate-safe view.
const (
	actionAccept   = "accept"
	actionStart    = "start"
	actionTakeover = "takeover"
	actionCancel   = "cancel"
)

// GetInterview assembles the candidate screen model for the resolved candidate session.
type GetInterview struct {
	Current     service.CurrentCandidateSession
	Catalog     service.NoticeCatalog
	Consents    candidateinterviewport.Consents
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
	Progress    service.InterviewProgress
	Clock       service.Clock
}

// Execute reads notice, consent, execution and progress from an allow-list; it never starts an interview or inserts consent.
func (q GetInterview) Execute(ctx context.Context, in GetInterviewInput) (*dto.InterviewView, error) {
	session, err := q.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	view := &dto.InterviewView{Actions: []string{}}
	notice, hasNotice := q.Catalog.Current(ctx)
	accepted := false
	if hasNotice {
		view.Notice = &dto.NoticeView{Version: notice.Version.String(), Statement: notice.Statement}
		consent, err := q.Consents.Find(ctx, session.AssignmentID, notice.Version)
		switch {
		case err == nil:
			accepted = true
			view.Consent = &dto.ConsentView{ID: consent.ID, NoticeVersion: consent.NoticeVersion.String(), AcceptedAt: consent.AcceptedAt}
		case !errors.Is(err, candidateinterviewport.ErrNotFound):
			return nil, err
		}
	}
	facts, found, err := q.execution(ctx, session.AssignmentID)
	if err != nil {
		return nil, err
	}
	if found {
		if err := q.fillExecution(ctx, view, facts, session.AssignmentID, in.ControllerID); err != nil {
			return nil, err
		}
		return view, nil
	}
	switch {
	case !hasNotice:
	case !accepted:
		view.Actions = append(view.Actions, actionAccept)
	case candidateinterview.SessionCovers(session.ExpiresAt, q.Clock.Now().Add(candidateinterview.InterviewBudget)):
		view.Actions = append(view.Actions, actionStart)
	}
	return view, nil
}

// execution reads the assignment's execution; absence, or a lifecycle adapter that is not bound yet, is no execution.
func (q GetInterview) execution(ctx context.Context, assignmentID string) (service.ExecutionFacts, bool, error) {
	facts, err := q.Lifecycle.Find(ctx, assignmentID)
	if errors.Is(err, service.ErrInterviewNotFound) || errors.Is(err, service.ErrInterviewDependencyUnavailable) {
		return service.ExecutionFacts{}, false, nil
	}
	return facts, err == nil, err
}

// fillExecution adds the safe execution, the controller-dependent action and live progress to the view.
func (q GetInterview) fillExecution(ctx context.Context, view *dto.InterviewView, facts service.ExecutionFacts, assignmentID, tab string) error {
	execution, err := safeExecution(ctx, q.Controllers, facts, assignmentID, tab)
	if err != nil {
		return err
	}
	view.Execution = execution
	switch facts.State {
	case "CREATED", "RUNNING":
		if execution.IsCurrentController {
			view.Actions = append(view.Actions, actionCancel)
		} else {
			view.Actions = append(view.Actions, actionTakeover)
		}
		view.Progress = q.progress(ctx, facts.ID)
	}
	return nil
}

// safeExecution builds the allow-listed execution view and marks whether the calling tab is the current controller.
func safeExecution(ctx context.Context, controllers candidateinterviewport.Controllers, facts service.ExecutionFacts, assignmentID, tab string) (*dto.InterviewExecutionView, error) {
	execution := &dto.InterviewExecutionView{ID: facts.ID, State: facts.State, Version: facts.Version, HardDeadline: copyTime(facts.HardDeadline)}
	controller, err := controllers.Find(ctx, assignmentID)
	if err != nil && !errors.Is(err, candidateinterviewport.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		execution.ControllerVersion = controller.Version
		if id, parseErr := candidateinterview.ParseControllerID(tab); parseErr == nil {
			execution.IsCurrentController = controller.ControllerID == id
		}
	}
	return execution, nil
}

// progress returns the candidate-safe progress, or nil when runtime has none or fails.
func (q GetInterview) progress(ctx context.Context, executionID string) *dto.ProgressView {
	if q.Progress == nil {
		return nil
	}
	current, err := q.Progress.Current(ctx, executionID)
	if err != nil || current == nil {
		return nil
	}
	return &dto.ProgressView{Phase: current.Phase, Connection: current.Connection}
}

var _ GetInterviewOperation = GetInterview{}

// copyTime returns a detached copy of an optional instant.
func copyTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	c := *t
	return &c
}
