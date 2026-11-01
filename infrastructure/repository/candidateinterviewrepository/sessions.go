package candidateinterviewrepository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"gorm.io/gorm"
)

// Sessions limits interview credentials to assignments whose access has not been revoked.
// Session lifetime and session revocation are still validated by the existing access authority.
type Sessions struct{ DB *gorm.DB }

var _ accessport.CandidateSessions = Sessions{}

// FindByTokenHash reads the access-owned session model joined to its assignment without mutating either.
func (s Sessions) FindByTokenHash(ctx context.Context, hash string) (accessport.CandidateSessionRecord, error) {
	var model accessrepository.CandidateSessionModel
	err := s.DB.WithContext(ctx).Select("candidate_sessions.*").
		Joins("JOIN interview_assignments ON interview_assignments.id = candidate_sessions.assignment_id").
		Where("candidate_sessions.token_hash = ? AND interview_assignments.access_revoked_at IS NULL", hash).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return accessport.CandidateSessionRecord{}, accessport.ErrNotFound
	}
	if err != nil {
		return accessport.CandidateSessionRecord{}, err
	}
	return accessport.CandidateSessionRecord{ID: model.ID, AssignmentID: model.AssignmentID, InvitationID: model.InvitationID, TokenHash: model.TokenHash, CreatedAt: model.CreatedAt, ExpiresAt: model.ExpiresAt, RevokedAt: model.RevokedAt}, nil
}
