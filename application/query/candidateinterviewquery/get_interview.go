package candidateinterviewquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// GetInterview assembles the candidate screen model for the resolved candidate session.
type GetInterview struct {
	Current     service.CurrentCandidateSession
	Catalog     service.NoticeCatalog
	Consents    candidateinterviewport.Consents
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
	Progress    service.InterviewProgress
	Clock       service.Clock
}

// Execute reads the screen model; this stub fails closed until the behavior is implemented.
func (q GetInterview) Execute(_ context.Context, _ GetInterviewInput) (*dto.InterviewView, error) {
	return nil, service.ErrInterviewDependencyUnavailable
}

var _ GetInterviewOperation = GetInterview{}
