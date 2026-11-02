package interviewusecase

import (
	"context"
	"strings"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// now normalizes injected trusted time at the application boundary.
func (u Workflow) now() time.Time { return u.Clock.Now().UTC().Truncate(time.Millisecond) }

// validAuthority rejects incomplete authority before any lifecycle or storage call.
func validAuthority(a service.InterviewAttempt) bool {
	return a.ExecutionID != "" && a.AttemptID != "" && a.ExpectedVersion > 0
}

// validSessionRequest checks only orchestration input, leaving candidate authentication to delivery.
func validSessionRequest(r SessionRequest) bool {
	return validAuthority(r.Authority) && r.SessionID != "" && r.IdempotencyKey != "" && strings.TrimSpace(r.SDP) != ""
}

// findCurrent rejects stale authority before reading and verifies the store's execution identity.
// This read check is not a fence for later writes; update always uses WithProgress.
func (u Workflow) findCurrent(ctx context.Context, a service.InterviewAttempt) (interviewport.ProgressRecord, error) {
	if !validAuthority(a) {
		return interviewport.ProgressRecord{}, interview.ErrInvalid
	}
	if err := u.Lifecycle.Fence(ctx, a); err != nil {
		return interviewport.ProgressRecord{}, err
	}
	r, err := u.Store.Find(ctx, a.ExecutionID)
	if err != nil {
		return r, err
	}
	if r.Snapshot.ExecutionID != a.ExecutionID {
		return interviewport.ProgressRecord{}, interview.ErrRestore
	}
	return r, nil
}

// update propagates the lifecycle transaction context to the versioned interview callback.
func (u Workflow) update(ctx context.Context, a service.InterviewAttempt, version int64, change func(*interviewport.ProgressRecord) error) error {
	return u.Lifecycle.WithProgress(ctx, a, func(tx context.Context) error {
		return u.Store.Update(tx, interviewport.ProgressWrite{ExecutionID: a.ExecutionID, AttemptID: a.AttemptID, ExpectedVersion: version}, func(r *interviewport.ProgressRecord) error {
			if r.Snapshot.ExecutionID != a.ExecutionID {
				return interview.ErrRestore
			}
			return change(r)
		})
	})
}

// closeSessions closes local shells as part of the caller's atomic progress mutation.
func closeSessions(r *interviewport.ProgressRecord, now time.Time) {
	for i := range r.Sessions {
		if r.Sessions[i].ClosedAt == nil {
			at := now
			r.Sessions[i].ClosedAt = &at
		}
	}
}

// releaseUnprovisioned closes the open shell left by a failed provider create and returns its id.
// A shell that already carries a provider identity, or an absent shell, is a conflict.
func releaseUnprovisioned(r *interviewport.ProgressRecord, now time.Time) (string, error) {
	for i := range r.Sessions {
		s := &r.Sessions[i]
		if s.ClosedAt != nil {
			continue
		}
		if s.ProviderSessionID != nil || s.Stage != r.Snapshot.CurrentStage {
			return "", interview.ErrConflict
		}
		at := now
		s.ClosedAt = &at
		return s.ID, nil
	}
	return "", interview.ErrConflict
}

// appendSession reserves a unique local session shell before any provider create call.
func appendSession(r *interviewport.ProgressRecord, req SessionRequest, now time.Time, replaced string) error {
	for _, s := range r.Sessions {
		if s.ID == req.SessionID || s.ClosedAt == nil {
			return interview.ErrConflict
		}
	}
	s := interviewport.VoiceSession{ID: req.SessionID, AttemptID: req.Authority.AttemptID, Stage: r.Snapshot.CurrentStage, OpenedAt: now}
	if replaced != "" {
		s.ReplacedSessionID = &replaced
	}
	r.Sessions = append(r.Sessions, s)
	return nil
}

// normalizeSDP preserves the prototype's CRLF and trailing-newline contract.
func normalizeSDP(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n") + "\r\n"
}
