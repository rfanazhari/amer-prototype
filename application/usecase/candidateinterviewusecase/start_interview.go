package candidateinterviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// StartInterview checks consent and coverage, then asks lifecycle to commit or find the execution.
type StartInterview struct {
	Current     service.CurrentCandidateSession
	Catalog     service.NoticeCatalog
	Consents    candidateinterviewport.Consents
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
	Clock       service.Clock
}

// Execute starts or finds the interview for the resolved candidate session.
func (u StartInterview) Execute(_ context.Context, _ StartInterviewInput) (*StartInterviewResult, error) {
	return nil, errNotImplemented
}

var _ StartInterviewOperation = StartInterview{}
