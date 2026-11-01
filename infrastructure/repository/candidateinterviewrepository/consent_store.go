// Package candidateinterviewrepository persists candidate-interview consent and controller records through GORM.
package candidateinterviewrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"gorm.io/gorm"
)

// ConsentStore implements the consent persistence ports through a runtime-owned GORM pool.
type ConsentStore struct {
	DB    *gorm.DB
	Clock service.Clock
}

// ConsentModel maps interview_consents independently of domain and API types.
type ConsentModel struct {
	ID                 string    `gorm:"type:uuid;primaryKey"`
	AssignmentID       string    `gorm:"column:assignment_id;type:uuid;not null"`
	NoticeVersion      string    `gorm:"column:notice_version;type:text;not null"`
	AcceptedAt         time.Time `gorm:"column:accepted_at;not null;autoCreateTime:false"`
	CandidateSessionID string    `gorm:"column:candidate_session_id;type:uuid;not null"`
}

// TableName fixes the consent table used by GORM.
func (ConsentModel) TableName() string { return "interview_consents" }

// txKey carries the open transaction through the context handed to WithAssignment callbacks.
type txKey struct{}

// db returns the transaction joined to ctx, or the pool bound to ctx.
func (s ConsentStore) db(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return s.DB.WithContext(ctx)
}

// WithAssignment locks the assignment row first, re-reads revocation and the candidate session, then runs fn in the transaction.
func (s ConsentStore) WithAssignment(ctx context.Context, assignmentID, sessionID string, fn func(context.Context) error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var revokedAt sql.NullTime
		err := tx.Raw("SELECT access_revoked_at FROM interview_assignments WHERE id = ? FOR UPDATE", assignmentID).Row().Scan(&revokedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return candidateinterviewport.ErrAssignmentUnavailable
		}
		if err != nil {
			return err
		}
		if revokedAt.Valid {
			return candidateinterviewport.ErrAssignmentUnavailable
		}
		var (
			owner            string
			expiresAt        time.Time
			sessionRevokedAt sql.NullTime
		)
		err = tx.Raw("SELECT assignment_id, expires_at, revoked_at FROM candidate_sessions WHERE id = ?", sessionID).Row().Scan(&owner, &expiresAt, &sessionRevokedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return candidateinterviewport.ErrAssignmentUnavailable
		}
		if err != nil {
			return err
		}
		if owner != assignmentID || sessionRevokedAt.Valid || !s.Clock.Now().UTC().Before(expiresAt) {
			return candidateinterviewport.ErrAssignmentUnavailable
		}
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Insert stores a new acceptance and maps a unique violation onto ErrAlreadyExists.
func (s ConsentStore) Insert(ctx context.Context, snap candidateinterview.ConsentSnapshot) error {
	model := ConsentModel{
		ID: snap.ID, AssignmentID: snap.AssignmentID, NoticeVersion: snap.NoticeVersion.String(),
		AcceptedAt: snap.AcceptedAt.UTC(), CandidateSessionID: snap.CandidateSessionID,
	}
	if err := s.db(ctx).Create(&model).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return candidateinterviewport.ErrAlreadyExists
		}
		return err
	}
	return nil
}

// Find returns the acceptance of one notice version for the assignment, or ErrNotFound.
func (s ConsentStore) Find(ctx context.Context, assignmentID string, version candidateinterview.NoticeVersion) (candidateinterview.ConsentSnapshot, error) {
	var model ConsentModel
	err := s.db(ctx).Where("assignment_id = ? AND notice_version = ?", assignmentID, version.String()).Take(&model).Error
	return restoreConsent(model, err)
}

// FindByID returns the acceptance with the id inside the assignment, or ErrNotFound.
func (s ConsentStore) FindByID(ctx context.Context, assignmentID, id string) (candidateinterview.ConsentSnapshot, error) {
	var model ConsentModel
	err := s.db(ctx).Where("id = ? AND assignment_id = ?", id, assignmentID).Take(&model).Error
	return restoreConsent(model, err)
}

// restoreConsent translates a read result into a snapshot; a corrupt stored row is a technical error.
func restoreConsent(model ConsentModel, err error) (candidateinterview.ConsentSnapshot, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return candidateinterview.ConsentSnapshot{}, candidateinterviewport.ErrNotFound
	}
	if err != nil {
		return candidateinterview.ConsentSnapshot{}, err
	}
	version, err := candidateinterview.ParseNoticeVersion(model.NoticeVersion)
	if err != nil {
		return candidateinterview.ConsentSnapshot{}, fmt.Errorf("stored consent %s: %w", model.ID, err)
	}
	consent, err := candidateinterview.RestoreConsent(candidateinterview.ConsentSnapshot{
		ID: model.ID, AssignmentID: model.AssignmentID, NoticeVersion: version,
		CandidateSessionID: model.CandidateSessionID, AcceptedAt: model.AcceptedAt.UTC(),
	})
	if err != nil {
		return candidateinterview.ConsentSnapshot{}, fmt.Errorf("stored consent %s: %w", model.ID, err)
	}
	return consent.Snapshot(), nil
}

var (
	_ candidateinterviewport.ConsentTransaction = ConsentStore{}
	_ candidateinterviewport.Consents           = ConsentStore{}
)
