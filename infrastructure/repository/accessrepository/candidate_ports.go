package accessrepository

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
)

// CandidateIdentities adapts Store onto the Candidates port without colliding with account FindByID.
type CandidateIdentities struct {
	Store Store
}

// Insert stores a new candidate identity.
func (s CandidateIdentities) Insert(ctx context.Context, record accessport.CandidateRecord) error {
	return s.Store.InsertCandidate(ctx, record)
}

// FindByID restores one candidate by identity or returns ErrNotFound.
func (s CandidateIdentities) FindByID(ctx context.Context, id string) (accessport.CandidateRecord, error) {
	return s.Store.FindCandidateByID(ctx, id)
}

// InvitationSecrets adapts Store onto the Invitations port without colliding with administrative session lookup.
type InvitationSecrets struct {
	Store Store
}

// FindByTokenHash restores one invitation by digest or returns ErrNotFound.
func (s InvitationSecrets) FindByTokenHash(ctx context.Context, tokenHash string) (accessport.InvitationRecord, error) {
	return s.Store.FindInvitationByTokenHash(ctx, tokenHash)
}

// CandidateSessionSecrets adapts Store onto the CandidateSessions port without colliding with administrative session lookup.
type CandidateSessionSecrets struct {
	Store Store
}

// FindByTokenHash restores one candidate session by digest or returns ErrNotFound.
func (s CandidateSessionSecrets) FindByTokenHash(ctx context.Context, tokenHash string) (accessport.CandidateSessionRecord, error) {
	return s.Store.FindCandidateSessionByTokenHash(ctx, tokenHash)
}
