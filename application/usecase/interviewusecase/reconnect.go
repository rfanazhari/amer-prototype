package interviewusecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// Disconnect stores backend detection only when the observation identifies the current open session.
// Opaque provider events never enter coverage or transcript through this operation.
func (u Workflow) Disconnect(ctx context.Context, a service.InterviewAttempt, s service.InterviewVoiceSession) error {
	if s.ExecutionID != a.ExecutionID || s.AttemptID != a.AttemptID {
		return interview.ErrConflict
	}
	record, err := u.findCurrent(ctx, a)
	if err != nil {
		return err
	}
	now := u.now()
	err = u.update(ctx, a, record.Version, func(r *interviewport.ProgressRecord) error {
		found := false
		for _, current := range r.Sessions {
			if current.ID == s.SessionID && current.AttemptID == a.AttemptID && current.ClosedAt == nil {
				found = true
			}
		}
		if !found {
			return interview.ErrConflict
		}
		p, err := interview.RestoreProgress(r.Snapshot)
		if err != nil {
			return err
		}
		if err = p.BeginReconnect(now); err != nil {
			return err
		}
		r.Snapshot = p.Snapshot()
		return nil
	})
	if errors.Is(err, interview.ErrBudget) {
		if failErr := u.Lifecycle.Fail(ctx, service.InterviewFailure{Authority: a, Reason: "reconnect_budget_exhausted"}); failErr != nil {
			return failErr
		}
	}
	return err
}

// Reconnect consumes stored recovery capacity and replaces a session on the same execution.
// A failed create retains detection and the consumed attempt for the next admitted retry.
func (u Workflow) Reconnect(ctx context.Context, req SessionRequest) (service.InterviewVoiceCreated, error) {
	if !validSessionRequest(req) {
		return service.InterviewVoiceCreated{}, interview.ErrInvalid
	}
	record, err := u.Store.Find(ctx, req.Authority.ExecutionID)
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	if record.Version != req.ProgressVersion {
		return service.InterviewVoiceCreated{}, interviewport.ErrConflict
	}
	p, err := interview.RestoreProgress(record.Snapshot)
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	now := u.now()
	if err = p.AttemptReconnect(now); err != nil {
		if errors.Is(err, interview.ErrBudget) && len(record.Sessions) > 0 {
			current := req.Authority
			current.AttemptID = record.Sessions[len(record.Sessions)-1].AttemptID
			if failErr := u.Lifecycle.Fail(ctx, service.InterviewFailure{Authority: current, Reason: "reconnect_budget_exhausted"}); failErr != nil {
				return service.InterviewVoiceCreated{}, failErr
			}
		}
		return service.InterviewVoiceCreated{}, err
	}
	if len(record.Sessions) == 0 {
		return service.InterviewVoiceCreated{}, interview.ErrConflict
	}
	replaced := record.Sessions[len(record.Sessions)-1].ID
	budgets := service.DefaultInterviewBudgets()
	if u.ProviderReady != nil && !u.ProviderReady() {
		return service.InterviewVoiceCreated{}, ErrProviderUnavailable
	}
	err = u.prepare(ctx, func(ctx context.Context) error {
		admitted, err := u.Lifecycle.Activate(ctx, service.InterviewActivation{Authority: req.Authority, Intent: "reconnect", IdempotencyKey: req.IdempotencyKey, DetectedAt: record.Snapshot.Reconnect.DetectedAt, Budgets: budgets})
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
		err = u.update(ctx, req.Authority, req.ProgressVersion, func(r *interviewport.ProgressRecord) error {
			agg, err := interview.RestoreProgress(r.Snapshot)
			if err != nil {
				return err
			}
			if err = agg.AttemptReconnect(now); err != nil {
				return err
			}
			r.Snapshot = agg.Snapshot()
			closeSessions(r, now)
			return appendSession(r, req, now, replaced)
		})
		if err != nil {
			return err
		}

		return u.reserve(ctx, req)
	})
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	text := recoveryContext(record, now)
	end := record.Snapshot.Reconnect.DetectedAt.Add(budgets.ReconnectWindow)
	hardDeadline := record.Snapshot.RunningAt.Add(15 * time.Minute)
	if hardDeadline.Before(end) {
		end = hardDeadline
	}
	recoveryCtx, cancel := context.WithTimeout(ctx, end.Sub(u.now()))
	defer cancel()
	return u.createSession(recoveryCtx, req, req.ProgressVersion+1, true, text)
}

// recoveryBoundary carries only the latest committed checkpoint and its final canonical entries.
type recoveryBoundary struct {
	CheckpointID string                      `json:"checkpointId"`
	Entries      []interview.TranscriptEntry `json:"entries"`
}

// recoveryContext excludes all uncited speech, configuration and provider memory.
func recoveryContext(r interviewport.ProgressRecord, now time.Time) string {
	var latest *interviewport.Checkpoint
	for i := range r.Checkpoints {
		c := &r.Checkpoints[i]
		if c.Stage == r.Snapshot.CurrentStage && !c.CommittedAt.After(now) && (latest == nil || c.CommittedAt.After(latest.CommittedAt)) {
			latest = c
		}
	}
	boundary := recoveryBoundary{}
	if latest != nil {
		boundary.CheckpointID = latest.ID
		for _, id := range latest.EntryIDs {
			for _, e := range r.Snapshot.Entries {
				if e.ID == id && e.State == interview.Final && e.Stage == latest.Stage {
					boundary.Entries = append(boundary.Entries, e)
				}
			}
		}
	}
	data, _ := json.Marshal(boundary)
	return string(data)
}
