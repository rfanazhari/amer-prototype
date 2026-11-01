package accessauthority

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// CandidateSessions resolves the current candidate session from a trusted context credential.
type CandidateSessions struct {
	Store  accessport.CandidateSessions
	Tokens service.TokenSecrets
	Clock  service.Clock
}

var _ service.CurrentCandidateSession = CandidateSessions{}

// Resolve binds a trusted candidate secret to a current unrevoked session or reports unbound access.
func (s CandidateSessions) Resolve(ctx context.Context) (accessport.CandidateSessionRecord, error) {
	plaintext, ok := service.CandidateCredentialFrom(ctx)
	if !ok {
		return accessport.CandidateSessionRecord{}, service.ErrCandidateSessionUnbound
	}
	digest, err := s.Tokens.Digest(service.CredentialCandidate, plaintext)
	if err != nil {
		return accessport.CandidateSessionRecord{}, service.ErrCandidateSessionUnbound
	}
	record, err := s.Store.FindByTokenHash(ctx, digest)
	if errors.Is(err, accessport.ErrNotFound) {
		return accessport.CandidateSessionRecord{}, service.ErrCandidateSessionUnbound
	}
	if err != nil {
		return accessport.CandidateSessionRecord{}, err
	}
	session, err := access.RestoreCandidateSession(access.CandidateSessionSnapshot{
		ID:           record.ID,
		AssignmentID: record.AssignmentID,
		InvitationID: record.InvitationID,
		CreatedAt:    record.CreatedAt,
		ExpiresAt:    record.ExpiresAt,
		RevokedAt:    record.RevokedAt,
	})
	if err != nil || !session.Current(trustedNow(s.Clock)) {
		return accessport.CandidateSessionRecord{}, service.ErrCandidateSessionUnbound
	}
	return record, nil
}
