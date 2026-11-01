package accessrepository

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
)

// CandidateSessionSecrets adapts Store onto the CandidateSessions port without colliding with administrative session lookup.
type CandidateSessionSecrets struct {
	Store Store
}

// FindByTokenHash currently reports not found until candidate session lookup is implemented.
func (CandidateSessionSecrets) FindByTokenHash(ctx context.Context, tokenHash string) (accessport.CandidateSessionRecord, error) {
	return accessport.CandidateSessionRecord{}, accessport.ErrNotFound
}
