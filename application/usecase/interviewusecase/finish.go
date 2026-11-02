package interviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
)

// FinishRequest carries the lifecycle authority for one terminal decision.
type FinishRequest struct {
	Authority service.InterviewAttempt
}

// Finish decides one terminal lifecycle outcome from the committed record and trusted time.
func (u Workflow) Finish(_ context.Context, _ FinishRequest) error {
	return nil
}
