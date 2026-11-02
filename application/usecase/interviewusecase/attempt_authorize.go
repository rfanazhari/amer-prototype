package interviewusecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// authorizeCandidate resolves the candidate session and proves the execution belongs to its assignment and
// the calling tab is the current controller. Another assignment's execution is not found, never forbidden.
func authorizeCandidate(ctx context.Context, current service.CurrentCandidateSession, executions service.ExecutionLifecycle,
	controllers candidateinterviewport.Controllers, executionID, controllerID string) (accessport.CandidateSessionRecord, service.ExecutionFacts, error) {
	tab, err := candidateinterview.ParseControllerID(controllerID)
	if err != nil {
		return accessport.CandidateSessionRecord{}, service.ExecutionFacts{}, ErrAttemptInvalid
	}
	session, err := current.Resolve(ctx)
	if err != nil {
		return session, service.ExecutionFacts{}, err
	}
	facts, err := executions.Find(ctx, session.AssignmentID)
	if err != nil {
		return session, facts, err
	}
	if facts.ID != executionID {
		return session, facts, service.ErrInterviewNotFound
	}
	held, err := controllers.Find(ctx, session.AssignmentID)
	switch {
	case errors.Is(err, candidateinterviewport.ErrNotFound):
		return session, facts, ErrAttemptConflict
	case err != nil:
		return session, facts, err
	case held.ControllerID != tab:
		return session, facts, ErrAttemptConflict
	}
	return session, facts, nil
}

// classify maps classified progress and lifecycle failures onto the attempt errors and leaves the rest untouched.
func classify(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrAttemptInvalid), errors.Is(err, ErrAttemptConflict), errors.Is(err, ErrAttemptNotReady), errors.Is(err, ErrProviderUnavailable):
		return err
	case errors.Is(err, interview.ErrInvalid):
		return ErrAttemptInvalid
	case errors.Is(err, interview.ErrConflict), errors.Is(err, interview.ErrBudget), errors.Is(err, interviewport.ErrConflict):
		return ErrAttemptConflict
	}
	return err
}

// validAttemptInput checks the operation's own preconditions before any port is called.
func validAttemptInput(in AttemptInput) bool {
	return (in.Intent == IntentActivate || in.Intent == IntentReconnect) && strings.TrimSpace(in.SDP) != "" &&
		in.IdempotencyKey != "" && in.ControllerID != "" && in.ExecutionID != "" && in.ExpectedVersion >= 1
}

// sessionCondition is ready only when the provider identity and browser readiness are both stored.
func sessionCondition(s interviewport.VoiceSession) string {
	if s.ProviderSessionID != nil && s.ReadinessAt != nil {
		return ConditionReady
	}
	return ConditionPending
}

// trusted returns the injected clock instant as UTC at millisecond precision.
func trusted(clock service.Clock) time.Time { return clock.Now().UTC().Truncate(time.Millisecond) }
