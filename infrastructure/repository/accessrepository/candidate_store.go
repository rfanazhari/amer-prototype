package accessrepository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"gorm.io/gorm"
)

// assignmentCollaborationTx holds assignment-first invitation and session writes.
type assignmentCollaborationTx struct {
	tx           *gorm.DB
	assignmentID string
}

// InsertCandidate stores a new candidate identity. Candidate email is not unique.
func (s Store) InsertCandidate(ctx context.Context, record accessport.CandidateRecord) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := fromCandidate(record)
		return tx.Create(&model).Error
	})
}

// FindCandidateByID restores one candidate by identity or returns ErrNotFound.
func (s Store) FindCandidateByID(ctx context.Context, id string) (accessport.CandidateRecord, error) {
	var model CandidateModel
	if err := s.DB.WithContext(ctx).Where("id = ?", id).Take(&model).Error; err != nil {
		return accessport.CandidateRecord{}, notFoundOrError(err)
	}
	return toCandidate(model)
}

// FindInvitationByTokenHash restores one invitation by digest or returns ErrNotFound.
func (s Store) FindInvitationByTokenHash(ctx context.Context, tokenHash string) (accessport.InvitationRecord, error) {
	var model InvitationModel
	if err := s.DB.WithContext(ctx).Where("token_hash = ?", tokenHash).Take(&model).Error; err != nil {
		return accessport.InvitationRecord{}, notFoundOrError(err)
	}
	return toInvitation(model)
}

// FindCandidateSessionByTokenHash restores one candidate session by digest or returns ErrNotFound.
func (s Store) FindCandidateSessionByTokenHash(ctx context.Context, tokenHash string) (accessport.CandidateSessionRecord, error) {
	var model CandidateSessionModel
	if err := s.DB.WithContext(ctx).Where("token_hash = ?", tokenHash).Take(&model).Error; err != nil {
		return accessport.CandidateSessionRecord{}, notFoundOrError(err)
	}
	return toCandidateSession(model)
}

// WithAssignment locks the Interview assignment row first, then runs access collaboration.
func (s Store) WithAssignment(ctx context.Context, assignmentID string, fn func(accessport.AssignmentTx) error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT id FROM interview_assignments WHERE id = ? FOR UPDATE", assignmentID).Error; err != nil {
			return err
		}
		return fn(assignmentCollaborationTx{tx: tx, assignmentID: assignmentID})
	})
}

// Facts returns Interview-owned access facts for the locked assignment.
func (t assignmentCollaborationTx) Facts(ctx context.Context) (accessport.AssignmentFacts, error) {
	var (
		id, candidateID, revisionID string
		revokedAt, completedAt      sql.NullTime
	)
	err := t.tx.WithContext(ctx).Raw(
		"SELECT id, candidate_id, published_revision_id, access_revoked_at, completed_at FROM interview_assignments WHERE id = ?",
		t.assignmentID,
	).Row().Scan(&id, &candidateID, &revisionID, &revokedAt, &completedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return accessport.AssignmentFacts{}, accessport.ErrNotFound
		}
		return accessport.AssignmentFacts{}, err
	}
	return accessport.AssignmentFacts{
		AssignmentID:      id,
		CandidateID:       candidateID,
		RevisionID:        revisionID,
		AccessRevoked:     revokedAt.Valid,
		TerminalCompleted: completedAt.Valid,
	}, nil
}

// CurrentInvitations lists unrevoked invitations for the locked assignment.
func (t assignmentCollaborationTx) CurrentInvitations(ctx context.Context) ([]accessport.InvitationRecord, error) {
	var models []InvitationModel
	if err := t.tx.WithContext(ctx).Where("assignment_id = ? AND revoked_at IS NULL", t.assignmentID).Find(&models).Error; err != nil {
		return nil, err
	}
	records := make([]accessport.InvitationRecord, 0, len(models))
	for _, model := range models {
		record, err := toInvitation(model)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// RevokeInvitation writes revoked_at explicitly for an invitation owned by the locked assignment.
func (t assignmentCollaborationTx) RevokeInvitation(ctx context.Context, id string, revokedAt *time.Time) error {
	return t.tx.WithContext(ctx).Model(&InvitationModel{}).Where("id = ? AND assignment_id = ?", id, t.assignmentID).
		Select("revoked_at").Updates(map[string]any{"revoked_at": copyTime(revokedAt)}).Error
}

// InsertInvitation stores a new invitation inside the locked assignment boundary.
func (t assignmentCollaborationTx) InsertInvitation(ctx context.Context, record accessport.InvitationRecord) error {
	model := fromInvitation(record)
	if err := t.tx.WithContext(ctx).Create(&model).Error; err != nil {
		return classifyUnique(err, accessport.TokenConflictError{})
	}
	return nil
}

// InsertCandidateSession stores a new candidate session inside the locked assignment boundary.
func (t assignmentCollaborationTx) InsertCandidateSession(ctx context.Context, record accessport.CandidateSessionRecord) error {
	model := fromCandidateSession(record)
	if err := t.tx.WithContext(ctx).Create(&model).Error; err != nil {
		return classifyUnique(err, accessport.TokenConflictError{})
	}
	return nil
}
