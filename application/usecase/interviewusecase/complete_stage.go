package interviewusecase

import (
	"context"

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
func (u Workflow) CompleteStage(ctx context.Context, req CompleteStageRequest) (StageOutcome, error) {
	return "", interview.ErrConflict
}
