package accessusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// IssueInvitation issues or replaces an assignment invitation under current HR authority.
type IssueInvitation struct {
	Assignments accessport.AssignmentAccess
	Current     service.CurrentAdministrativeSession
	Tokens      service.TokenSecrets
	Clock       service.Clock
}

// Execute persists a new invitation after assignment-first eligibility checks and returns the plaintext token once.
func (u IssueInvitation) Execute(ctx context.Context, assignmentID string) (*dto.IssuedInvitation, error) {
	return nil, errors.New("issue invitation is not implemented")
}
