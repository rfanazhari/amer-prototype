package interviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// StageOutcome names what a stage-completion statement did.
type StageOutcome string

const (
	// StageAdvanced means the stage was sealed and the next stage is current.
	StageAdvanced StageOutcome = "advanced"
	// StageRepeated means the stage was already sealed and nothing changed.
	StageRepeated StageOutcome = "repeated"
	// InterviewCompleted means the closing stage was sealed and the lifecycle completed the interview.
	InterviewCompleted StageOutcome = "completed"
)

// CompleteStageRequest carries the lifecycle authority and the stage the model says is finished.
type CompleteStageRequest struct {
	Authority service.InterviewAttempt
	Stage     enum.QuestionStage
}

// CompleteStage applies the model's statement that a stage is finished.
// The model only states it: order, the stage clock and the hard deadline are checked here, and no coverage is
// needed because coverage is analysed after the interview. The statement seals the current stage and makes the
// next one current in the same session. A statement for an earlier stage is a no-op. The closing statement seals
// closing, closes the local session records and asks the lifecycle to complete; repeating it retries completion
// only, so a lifecycle failure after the commit can be finished by the next statement.
func (u Workflow) CompleteStage(ctx context.Context, req CompleteStageRequest) (StageOutcome, error) {
	if !req.Stage.Valid() {
		return "", interview.ErrInvalid
	}
	record, err := u.findCurrent(ctx, req.Authority)
	if err != nil {
		return "", err
	}
	p, err := interview.RestoreProgress(record.Snapshot)
	if err != nil {
		return "", err
	}
	current, want := stageRank(record.Snapshot.CurrentStage), stageRank(req.Stage)
	switch {
	case want < current:
		return StageRepeated, nil
	case want > current:
		return "", interview.ErrConflict
	case p.AllStagesSealed():
		return u.completeInterview(ctx, req, record.Sessions)
	}
	if err = p.Attest(u.now()); err != nil {
		return "", err
	}
	closing := req.Stage == enum.StageClosing
	err = u.update(ctx, req.Authority, record.Version, func(r *interviewport.ProgressRecord) error {
		agg, restoreErr := interview.RestoreProgress(r.Snapshot)
		if restoreErr != nil {
			return restoreErr
		}
		committedAt := u.now()
		if attestErr := agg.Attest(committedAt); attestErr != nil {
			return attestErr
		}
		r.Snapshot = agg.Snapshot()
		if closing {
			closeSessions(r, committedAt)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if closing {
		return u.completeInterview(ctx, req, record.Sessions)
	}
	return StageAdvanced, nil
}

// completeInterview asks the lifecycle to complete once the closing stage is sealed, then ends the provider session.
// The provider close is best effort: it cannot undo the committed completion.
func (u Workflow) completeInterview(ctx context.Context, req CompleteStageRequest, sessions []interviewport.VoiceSession) (StageOutcome, error) {
	if err := u.Lifecycle.Complete(ctx, service.InterviewCompletion{Authority: req.Authority, ProgressReference: req.Authority.ExecutionID}); err != nil {
		return "", err
	}
	u.closeProvider(ctx, req.Authority.ExecutionID, sessions)
	return InterviewCompleted, nil
}

// closeProvider ends the latest session that holds a stored provider id, ignoring failure after the commit.
func (u Workflow) closeProvider(ctx context.Context, executionID string, sessions []interviewport.VoiceSession) {
	if u.Driver == nil {
		return
	}
	for i := len(sessions) - 1; i >= 0; i-- {
		s := sessions[i]
		if s.ProviderSessionID == nil || *s.ProviderSessionID == "" {
			continue
		}
		u.closeCreated(ctx, service.InterviewVoiceSession{ExecutionID: executionID, AttemptID: s.AttemptID, SessionID: s.ID, ProviderSessionID: *s.ProviderSessionID})
		return
	}
}

// stageRank orders the four fixed stages; an unknown stage ranks after them.
func stageRank(stage enum.QuestionStage) int {
	for i, s := range interviewStages {
		if s == stage {
			return i
		}
	}
	return len(interviewStages)
}

// CompleteInterview seals every remaining stage in order and completes the interview on the candidate's finish.
// The attempt is the latest session's, so a repeat after a partial run resumes from the stored stage. Each statement
// is its own committed step: a failure leaves the sealed stages in place and the same call can be repeated.
func (u Workflow) CompleteInterview(ctx context.Context, executionID string, expectedVersion int64) error {
	record, err := u.Store.Find(ctx, executionID)
	if err != nil {
		return err
	}
	if record.Snapshot.ExecutionID != executionID || len(record.Sessions) == 0 {
		return interview.ErrConflict
	}
	authority := service.InterviewAttempt{ExecutionID: executionID, AttemptID: record.Sessions[len(record.Sessions)-1].AttemptID, ExpectedVersion: expectedVersion}
	for _, stage := range interviewStages[stageRank(record.Snapshot.CurrentStage):] {
		if _, err = u.CompleteStage(ctx, CompleteStageRequest{Authority: authority, Stage: stage}); err != nil {
			return err
		}
	}
	return nil
}
