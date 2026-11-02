package interviewusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Ready records browser readiness on the current attempt of the session's own execution.
type Ready struct {
	Current     service.CurrentCandidateSession
	Executions  service.ExecutionLifecycle
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.InterviewLifecycle
	Progress    interviewport.ProgressStore
	Clock       service.Clock
}

var _ ReadyOperation = Ready{}

// Execute is not implemented yet.
func (u Ready) Execute(context.Context, ReadyInput) (*ReadyResult, error) {
	return nil, errors.New("not implemented")
}
