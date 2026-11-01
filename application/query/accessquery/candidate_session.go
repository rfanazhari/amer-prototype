package accessquery

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// CandidateSession returns the current candidate session after trusted assignment binding.
type CandidateSession struct {
	Current     service.CurrentCandidateSession
	Candidates  accessport.Candidates
	Assignments accessport.AssignmentAccess
	CSRF        service.CSRFTokens
	Clock       service.Clock
}

// Execute returns detached candidate, assignment and session material without creating execution.
func (q CandidateSession) Execute(ctx context.Context) (*dto.CandidateAccess, error) {
	return nil, errors.New("candidate session is not implemented")
}
