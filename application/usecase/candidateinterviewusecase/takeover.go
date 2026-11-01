package candidateinterviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Takeover replaces the controlling tab of the assignment's live interview.
type Takeover struct {
	Current     service.CurrentCandidateSession
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
	Clock       service.Clock
}

// Execute makes the calling tab the controller.
func (u Takeover) Execute(_ context.Context, _ TakeoverInput) (*dto.TakeoverView, error) {
	return nil, errNotImplemented
}

var _ TakeoverOperation = Takeover{}
