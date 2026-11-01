package accessrepository_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

// candidatePersistCase names one candidate or assignment-access SQL behavior.
type candidatePersistCase struct {
	name string
	run  func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock)
}

// TestCandidatePersistenceSQL covers candidate identity and assignment-scoped invitation/session SQL without AutoMigrate.
func TestCandidatePersistenceSQL(t *testing.T) {
	candidate := testdata.CandidateRecord(t)
	invitation := testdata.InvitationRecord(t)
	session := testdata.CandidateSessionRecord(t)
	revoked := testdata.Now().Add(time.Minute)
	facts := testdata.AssignmentFacts(t)
	tests := []candidatePersistCase{
		{
			name: "candidate insert writes nonunique contact email",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)INSERT INTO ["']?candidates["']?`).WithArgs(
					candidate.ID, candidate.Name, candidate.Email, candidate.CreatedBy, candidate.CreatedAt, candidate.UpdatedAt,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				require.NoError(t, store.InsertCandidate(context.Background(), candidate))
			},
		},
		{
			name: "candidate find restores identity by id",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "email", "created_by", "created_at", "updated_at"}).
					AddRow(candidate.ID, candidate.Name, candidate.Email, candidate.CreatedBy, candidate.CreatedAt, candidate.UpdatedAt)
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?candidates["']?`).WithArgs(candidate.ID, 1).WillReturnRows(rows)
				got, err := store.FindCandidateByID(context.Background(), candidate.ID)
				require.NoError(t, err)
				require.Equal(t, candidate.ID, got.ID)
				require.Equal(t, candidate.Email, got.Email)
			},
		},
		{
			name: "assignment lock precedes invitation insert and former revocation",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)SELECT id FROM interview_assignments WHERE id = .* FOR UPDATE`).WithArgs(testdata.AssignmentID).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_assignments["']?`).WillReturnRows(
					sqlmock.NewRows([]string{"id", "candidate_id", "published_revision_id", "access_revoked_at", "completed_at"}).
						AddRow(facts.AssignmentID, facts.CandidateID, facts.RevisionID, nil, nil),
				)
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_invitations["']?`).WillReturnRows(
					sqlmock.NewRows([]string{"id", "assignment_id", "token_hash", "created_by", "created_at", "expires_at", "revoked_at"}).
						AddRow(invitation.ID, invitation.AssignmentID, invitation.TokenHash, invitation.CreatedBy, invitation.CreatedAt, invitation.ExpiresAt, nil),
				)
				mock.ExpectExec(`(?i)UPDATE ["']?interview_invitations["']? SET .*revoked_at`).WithArgs(revoked, invitation.ID).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`(?i)INSERT INTO ["']?interview_invitations["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				replacement := invitation
				replacement.ID = "cccccccc-cccc-4ccc-8ccc-ccccccccccc2"
				err := store.WithAssignment(context.Background(), testdata.AssignmentID, func(tx accessport.AssignmentTx) error {
					got, err := tx.Facts(context.Background())
					require.NoError(t, err)
					require.Equal(t, facts.AssignmentID, got.AssignmentID)
					current, err := tx.CurrentInvitations(context.Background())
					require.NoError(t, err)
					require.Equal(t, invitation.ID, current[0].ID)
					require.NoError(t, tx.RevokeInvitation(context.Background(), invitation.ID, &revoked))
					return tx.InsertInvitation(context.Background(), replacement)
				})
				require.NoError(t, err)
			},
		},
		{
			name: "candidate session insert is assignment scoped",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)SELECT id FROM interview_assignments WHERE id = .* FOR UPDATE`).WithArgs(testdata.AssignmentID).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`(?i)INSERT INTO ["']?candidate_sessions["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				err := store.WithAssignment(context.Background(), testdata.AssignmentID, func(tx accessport.AssignmentTx) error {
					return tx.InsertCandidateSession(context.Background(), session)
				})
				require.NoError(t, err)
			},
		},
		{
			name: "invitation lookup uses digest namespace",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "assignment_id", "token_hash", "created_by", "created_at", "expires_at", "revoked_at"}).
					AddRow(invitation.ID, invitation.AssignmentID, invitation.TokenHash, invitation.CreatedBy, invitation.CreatedAt, invitation.ExpiresAt, nil)
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_invitations["']?`).WithArgs(invitation.TokenHash, 1).WillReturnRows(rows)
				got, err := store.FindInvitationByTokenHash(context.Background(), invitation.TokenHash)
				require.NoError(t, err)
				require.Equal(t, invitation.AssignmentID, got.AssignmentID)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}
