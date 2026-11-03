package interviewusecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// Attempts creates or replays a candidate voice attempt for the session's own execution.
type Attempts struct {
	Current     service.CurrentCandidateSession
	Executions  service.ExecutionLifecycle
	Controllers candidateinterviewport.Controllers
	Commands    interviewport.SessionCommands
	Revisions   interviewport.RevisionReader
	Progress    interviewport.ProgressStore
	Sessions    SessionStarter
	Clock       service.Clock
	NewID       func() string
}

var _ AttemptOperation = Attempts{}

// Execute authorizes the candidate, replays a stored command for the same key and intent, and otherwise opens one
// attempt through the session starter. Replay is checked before the version so a repeated request after success
// does not conflict with the version its own attempt advanced. Workflow reserves and finalizes its command atomically.
func (u Attempts) Execute(ctx context.Context, in AttemptInput) (*AttemptResult, error) {
	if !validAttemptInput(in) {
		return nil, ErrAttemptInvalid
	}
	session, facts, err := authorizeCandidate(ctx, u.Current, u.Executions, u.Controllers, in.ExecutionID, in.ControllerID)
	if err != nil {
		return nil, err
	}
	key := session.ID + ":" + in.IdempotencyKey
	stored, err := u.Commands.Find(ctx, in.ExecutionID, key)
	switch {
	case err == nil:
		return u.replay(ctx, in, stored)
	case !errors.Is(err, interviewport.ErrNotFound):
		return nil, err
	}
	if facts.Version != in.ExpectedVersion {
		return nil, ErrAttemptConflict
	}
	req, err := u.request(ctx, in, key)
	if err != nil {
		return nil, err
	}
	created, err := u.start(ctx, in.Intent, req)
	if err != nil {
		if errors.Is(classify(err), ErrAttemptConflict) {
			if stored, findErr := u.Commands.Find(ctx, in.ExecutionID, key); findErr == nil {
				return u.replay(ctx, in, stored)
			}
		}
		return nil, classify(err)
	}
	return &AttemptResult{AttemptID: created.Session.AttemptID, SessionID: created.Session.SessionID, SDP: created.SDP, ExecutionVersion: created.ExecutionVersion, SessionCondition: ConditionPending}, nil
}

// request builds the workflow request, minting the attempt and session identities and reading the progress version
// and, for activation, the pinned revision.
func (u Attempts) request(ctx context.Context, in AttemptInput, key string) (SessionRequest, error) {
	req := SessionRequest{
		Authority: service.InterviewAttempt{ExecutionID: in.ExecutionID, AttemptID: u.id(), ExpectedVersion: in.ExpectedVersion},
		SessionID: u.id(), IdempotencyKey: key, SDP: in.SDP,
	}
	req.Command = &interviewport.SessionCommand{ID: u.id(), ExecutionID: in.ExecutionID, Key: key, Intent: in.Intent, AttemptID: req.Authority.AttemptID, SessionID: req.SessionID, State: "pending", CreatedAt: trusted(u.Clock)}
	req.Check = func(tx context.Context) error {
		_, _, err := authorizeCandidate(tx, u.Current, u.Executions, u.Controllers, in.ExecutionID, in.ControllerID)
		return err
	}
	record, err := u.Progress.Find(ctx, in.ExecutionID)
	switch {
	case err == nil:
		req.ProgressVersion = record.Version
		if in.Intent == IntentActivate {
			for _, session := range record.Sessions {
				if session.ProviderSessionID != nil {
					return req, ErrAttemptConflict
				}
			}
		}
	case !errors.Is(err, interviewport.ErrNotFound):
		return req, err
	}
	if in.Intent == IntentActivate {
		if req.Revision, err = u.Revisions.Find(ctx, in.ExecutionID); err != nil {
			return req, err
		}
	}
	return req, nil
}

// start dispatches to the starter operation that matches the intent.
func (u Attempts) start(ctx context.Context, intent string, req SessionRequest) (service.InterviewVoiceCreated, error) {
	if intent == IntentReconnect {
		return u.Sessions.Reconnect(ctx, req)
	}
	return u.Sessions.Activate(ctx, req)
}

// replay returns the stored attempt and original answer for the same intent; another intent under the key conflicts.
func (u Attempts) replay(ctx context.Context, in AttemptInput, stored interviewport.SessionCommand) (*AttemptResult, error) {
	if stored.Intent != in.Intent || stored.State == "failed" || stored.State == "legacy" {
		return nil, ErrAttemptConflict
	}
	if stored.State == "pending" {
		return nil, ErrAttemptNotReady
	}
	if stored.State != "succeeded" || stored.SessionID == "" || stored.AnswerSDP == "" || stored.ExecutionVersion < 1 {
		return nil, interview.ErrRestore
	}
	condition := ConditionPending
	record, err := u.Progress.Find(ctx, in.ExecutionID)
	switch {
	case err == nil:
		for _, s := range record.Sessions {
			if s.ID == stored.SessionID {
				condition = sessionCondition(s)
			}
		}
	case !errors.Is(err, interviewport.ErrNotFound):
		return nil, err
	}
	return &AttemptResult{AttemptID: stored.AttemptID, SessionID: stored.SessionID, SDP: stored.AnswerSDP, ExecutionVersion: stored.ExecutionVersion, SessionCondition: condition, Replayed: true}, nil
}

// id returns the injected identity minter result, or a random UUID when none is injected.
func (u Attempts) id() string {
	if u.NewID != nil {
		return u.NewID()
	}
	return uuid.NewString()
}
