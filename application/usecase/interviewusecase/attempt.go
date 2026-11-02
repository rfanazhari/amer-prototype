package interviewusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
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

// Execute is not implemented yet.
func (u Attempts) Execute(context.Context, AttemptInput) (*AttemptResult, error) {
	return nil, errors.New("not implemented")
}
