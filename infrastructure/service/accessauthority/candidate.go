package accessauthority

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// CandidateSessions resolves the current candidate session from a trusted context credential.
type CandidateSessions struct {
	Store  accessport.CandidateSessions
	Tokens service.TokenSecrets
	Clock  service.Clock
}

var _ service.CurrentCandidateSession = CandidateSessions{}

// Resolve currently reports unbound access until candidate cookie lookup is implemented.
func (CandidateSessions) Resolve(ctx context.Context) (accessport.CandidateSessionRecord, error) {
	return accessport.CandidateSessionRecord{}, service.ErrCandidateSessionUnbound
}
