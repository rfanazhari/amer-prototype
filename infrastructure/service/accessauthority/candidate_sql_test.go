package accessauthority_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

// candidateSQLSessionCase names one SQL-backed candidate session resolution outcome.
type candidateSQLSessionCase struct {
	name     string
	expires  time.Time
	revoked  *time.Time
	noCookie bool
	noRows   bool
	wantErr  error
}

// TestCandidateSessionsResolveSQL uses sqlmock to prove current, expired and missing candidate cookies.
func TestCandidateSessionsResolveSQL(t *testing.T) {
	revokedAt := testdata.Now().Add(time.Minute)
	tests := []candidateSQLSessionCase{
		{name: "current candidate session", expires: testdata.Now().Add(2 * time.Hour)},
		{name: "expired candidate session", expires: testdata.Now().Add(time.Hour), wantErr: service.ErrCandidateSessionUnbound},
		{name: "revoked candidate session", expires: testdata.Now().Add(2 * time.Hour), revoked: &revokedAt, wantErr: service.ErrCandidateSessionUnbound},
		{name: "unknown digest", expires: testdata.Now().Add(2 * time.Hour), noRows: true, wantErr: service.ErrCandidateSessionUnbound},
		{name: "missing cookie", expires: testdata.Now().Add(2 * time.Hour), noCookie: true, wantErr: service.ErrCandidateSessionUnbound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tokens := mocks.NewTokenSecrets(t)
			clock := mocks.NewClock(t)
			clock.On("Now").Return(testdata.Now().Add(time.Hour)).Maybe()
			if !tt.noCookie {
				tokens.On("Digest", service.CredentialCandidate, "plain-candidate-secret").Return(testdata.TokenHash, nil)
				expectCandidateSessionQuery(t, mock, tt.expires, tt.revoked, tt.noRows)
			}
			resolver := accessauthority.CandidateSessions{Store: accessrepository.CandidateSessionSecrets{Store: store}, Tokens: tokens, Clock: clock}
			ctx := context.Background()
			if !tt.noCookie {
				ctx = service.WithCandidateCredential(ctx, "plain-candidate-secret")
			}
			got, err := resolver.Resolve(ctx)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testdata.CandidateSessionID, got.ID)
			require.Equal(t, testdata.AssignmentID, got.AssignmentID)
		})
	}
}

// expectCandidateSessionQuery prepares candidate_sessions lookup used by the resolver.
func expectCandidateSessionQuery(t *testing.T, mock sqlmock.Sqlmock, expires time.Time, revoked *time.Time, noRows bool) {
	t.Helper()
	columns := []string{"id", "assignment_id", "invitation_id", "token_hash", "created_at", "expires_at", "revoked_at"}
	if noRows {
		mock.ExpectQuery(`(?i)SELECT .* FROM .*candidate_sessions`).WithArgs(testdata.TokenHash, 1).WillReturnRows(sqlmock.NewRows(columns))
		return
	}
	session := testdata.CandidateSessionRecord(t)
	mock.ExpectQuery(`(?i)SELECT .* FROM .*candidate_sessions`).WithArgs(testdata.TokenHash, 1).WillReturnRows(
		sqlmock.NewRows(columns).AddRow(session.ID, session.AssignmentID, session.InvitationID, testdata.TokenHash, session.CreatedAt, expires, revoked),
	)
}
