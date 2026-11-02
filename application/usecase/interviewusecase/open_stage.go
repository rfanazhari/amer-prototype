package interviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// OpenStage reserves a successor shell only after reading a previously committed handoff.
// A repeated call after a failed provider create retries the committed target only: the unprovisioned
// shell is closed and replaced. It performs no retry loop and assigns no attempt budget (T16).
func (u Workflow) OpenStage(ctx context.Context, req SessionRequest) (service.InterviewVoiceCreated, error) {
	if !validSessionRequest(req) {
		return service.InterviewVoiceCreated{}, interview.ErrInvalid
	}
	record, err := u.findCurrent(ctx, req.Authority)
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	if len(record.Snapshot.Handoffs) == 0 {
		return service.InterviewVoiceCreated{}, interview.ErrConflict
	}
	target := record.Snapshot.Handoffs[len(record.Snapshot.Handoffs)-1].Target
	if target == "" {
		return service.InterviewVoiceCreated{}, interview.ErrConflict
	}
	now := u.now()
	err = u.update(ctx, req.Authority, req.ProgressVersion, func(r *interviewport.ProgressRecord) error {
		if r.Snapshot.CurrentStage == target {
			replaced, err := releaseUnprovisioned(r, now)
			if err != nil {
				return err
			}
			return appendSession(r, req, now, replaced)
		}
		p, err := interview.RestoreProgress(r.Snapshot)
		if err != nil {
			return err
		}
		if err = p.ActivateStage(target, now); err != nil {
			return err
		}
		r.Snapshot = p.Snapshot()
		return appendSession(r, req, now, "")
	})
	if err != nil {
		return service.InterviewVoiceCreated{}, err
	}
	return u.createSession(ctx, req, req.ProgressVersion+1, false, "")
}
