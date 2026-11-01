package accessrepository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
)

// InsertCandidate stores a new candidate identity. Candidate email is not unique.
func (s Store) InsertCandidate(ctx context.Context, record accessport.CandidateRecord) error {
	return errors.New("candidate insert is not implemented")
}

// FindCandidateByID restores one candidate by identity or returns ErrNotFound.
func (s Store) FindCandidateByID(ctx context.Context, id string) (accessport.CandidateRecord, error) {
	return accessport.CandidateRecord{}, errors.New("candidate find is not implemented")
}

// FindInvitationByTokenHash restores one invitation by digest or returns ErrNotFound.
func (s Store) FindInvitationByTokenHash(ctx context.Context, tokenHash string) (accessport.InvitationRecord, error) {
	return accessport.InvitationRecord{}, errors.New("invitation find is not implemented")
}

// WithAssignment locks the Interview assignment row first, then runs access collaboration.
func (s Store) WithAssignment(ctx context.Context, assignmentID string, fn func(accessport.AssignmentTx) error) error {
	return errors.New("assignment collaboration is not implemented")
}
