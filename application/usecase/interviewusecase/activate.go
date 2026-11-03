package interviewusecase

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// Activate asks the lifecycle to enforce persisted initial budgets before creating a session.
// Replay delivery and candidate authorization belong to T8; a replay never creates another session here.
func (u Workflow) Activate(ctx context.Context, req SessionRequest) (service.InterviewVoiceCreated, error) {
	if !validSessionRequest(req) {
		return service.InterviewVoiceCreated{}, interview.ErrInvalid
	}
	budgets := service.DefaultInterviewBudgets()
	if u.ProviderReady != nil && !u.ProviderReady() {
		return service.InterviewVoiceCreated{}, ErrProviderUnavailable
	}
	version := int64(1)
	var setupDeadline time.Time
	err := u.prepare(ctx, func(ctx context.Context) error {
		admitted, err := u.Lifecycle.Activate(ctx, service.InterviewActivation{Authority: req.Authority, Intent: "activate", IdempotencyKey: req.IdempotencyKey, Budgets: budgets})
		if err != nil {
			return err
		}
		if admitted.Replayed {
			return interview.ErrConflict
		}
		if err = validateAdmission(req, admitted); err != nil {
			return err
		}
		req.Authority = admitted.Authority
		if req.Check != nil {
			if err = req.Check(ctx); err != nil {
				return err
			}
		}
		record, findErr := u.Store.Find(ctx, req.Authority.ExecutionID)
		version = int64(1)
		now := u.now()
		setupDeadline = now.Add(budgets.ActivationAttemptTimeout)
		for _, limit := range []time.Time{admitted.RunningAt.Add(budgets.ActivationWindow), admitted.HardDeadline} {
			if limit.Before(setupDeadline) {
				setupDeadline = limit
			}
		}
		if now.Before(admitted.RunningAt) || !now.Before(admitted.HardDeadline) {
			return interview.ErrBudget
		}
		switch {
		case errors.Is(findErr, interviewport.ErrNotFound):
			p, progressErr := interview.NewProgress(req.Authority.ExecutionID, req.Revision, admitted.RunningAt)
			if progressErr != nil {
				return progressErr
			}
			record = interviewport.ProgressRecord{Snapshot: p.Snapshot(), Version: 1}
			if err = appendSession(&record, req, now, ""); err != nil {
				return err
			}
			err = u.Lifecycle.WithProgress(ctx, req.Authority, func(tx context.Context) error {
				if err := u.Store.Insert(tx, interviewport.ProgressWrite{ExecutionID: req.Authority.ExecutionID, AttemptID: req.Authority.AttemptID}, record); err != nil {
					return err
				}
				return u.reserve(tx, req)
			})
		case findErr != nil:
			return findErr
		default:
			err = u.update(ctx, req.Authority, req.ProgressVersion, func(r *interviewport.ProgressRecord) error {
				if r.Snapshot.CurrentStage != enum.StageBackground || len(r.Snapshot.Handoffs) != 0 || !r.Snapshot.RunningAt.Equal(admitted.RunningAt) {
					return interview.ErrConflict
				}
				for _, session := range r.Sessions {
					if session.ProviderSessionID != nil {
						return interview.ErrConflict
					}
				}
				closeSessions(r, now)
				return appendSession(r, req, now, "")
			})
			if err == nil {
				err = u.reserve(ctx, req)
			}
			version = req.ProgressVersion + 1
		}
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	createCtx, cancel := context.WithTimeout(ctx, setupDeadline.Sub(u.now()))
	defer cancel()
	return u.createSession(createCtx, req, version, false, "")
}

// validateAdmission ensures the lifecycle cannot accidentally redirect this interview to another execution.
func validateAdmission(req SessionRequest, a service.InterviewActivationResult) error {
	if !validAuthority(a.Authority) || a.Authority.ExecutionID != req.Authority.ExecutionID || a.RunningAt.IsZero() || !a.HardDeadline.Equal(a.RunningAt.Add(15*time.Minute)) {
		return interview.ErrConflict
	}
	return nil
}
