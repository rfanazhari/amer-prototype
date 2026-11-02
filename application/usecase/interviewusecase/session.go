package interviewusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// createSession runs only after the session shell commits and persists provider identity under authority.
// Provider failure never rolls back an earlier handoff. No retry is performed here.
func (u Workflow) createSession(ctx context.Context, req SessionRequest, version int64, reconnect bool, text string) (service.InterviewVoiceCreated, error) {
	session := service.InterviewVoiceSession{ExecutionID: req.Authority.ExecutionID, AttemptID: req.Authority.AttemptID, SessionID: req.SessionID}
	committed, err := u.Store.Find(ctx, req.Authority.ExecutionID)
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	instructions, err := u.stageInstructions(ctx, committed.Snapshot)
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	created, err := u.Driver.Create(ctx, service.InterviewVoiceCreate{Session: session, SDP: normalizeSDP(req.SDP)})
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	if created.Session.ExecutionID != session.ExecutionID || created.Session.AttemptID != session.AttemptID || created.Session.SessionID != session.SessionID || created.Session.ProviderSessionID == "" || created.SDP == "" {
		u.closeCreated(ctx, created.Session)
		return service.InterviewVoiceCreated{}, interview.ErrConflict
	}
	if err = u.Driver.Attach(ctx, created.Session); err == nil {
		err = u.Driver.SupplyContext(ctx, service.InterviewVoiceContext{Session: created.Session, Text: instructions})
	}
	if err == nil && reconnect {
		err = u.Driver.SupplyContext(ctx, service.InterviewVoiceContext{Session: created.Session, Text: text})
	}
	if err != nil {
		u.closeCreated(ctx, created.Session)
		return service.InterviewVoiceCreated{}, err
	}
	err = u.update(ctx, req.Authority, version, func(r *interviewport.ProgressRecord) error {
		found := false
		for i := range r.Sessions {
			s := &r.Sessions[i]
			if s.ID == session.SessionID && s.AttemptID == session.AttemptID && s.ClosedAt == nil {
				id := created.Session.ProviderSessionID
				s.ProviderSessionID = &id
				found = true
			}
		}
		if !found {
			return interview.ErrConflict
		}
		if reconnect {
			p, err := interview.RestoreProgress(r.Snapshot)
			if err != nil {
				return err
			}
			if err = p.FinishReconnect(); err != nil {
				return err
			}
			r.Snapshot = p.Snapshot()
		}
		return nil
	})
	if err != nil {
		u.closeCreated(ctx, created.Session)
		return service.InterviewVoiceCreated{}, err
	}
	return created, nil
}

// closeCreated bounds cleanup even if the original request was cancelled.
// A close failure cannot undo committed progress or replace the primary error.
func (u Workflow) closeCreated(ctx context.Context, s service.InterviewVoiceSession) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = u.Driver.Close(cleanup, s)
}
