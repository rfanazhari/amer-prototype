package accessquery_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// candidateSessionQueryCase names one current candidate-session reload outcome.
type candidateSessionQueryCase struct {
	name       string
	session    accessport.CandidateSessionRecord
	facts      accessport.AssignmentFacts
	now        time.Time
	resolveErr error
	wantErr    error
}

// TestCandidateSessionReloadReusesAccessWithoutCreatingExecution covers expiry, revocation and cross-assignment denial.
func TestCandidateSessionReloadReusesAccessWithoutCreatingExecution(t *testing.T) {
	session := testdata.CandidateSessionRecord(t)
	live := testdata.AssignmentFacts(t)
	other := testdata.AssignmentFacts(t)
	other.AssignmentID = "ffffffff-ffff-4fff-8fff-fffffffffff1"
	revoked := testdata.AssignmentFacts(t)
	revoked.AccessRevoked = true
	tests := []candidateSessionQueryCase{
		{name: "current session reloads", session: session, facts: live, now: testdata.Now().Add(time.Minute)},
		{name: "exact expiry refused", session: session, facts: live, now: testdata.Now().Add(2 * time.Hour), wantErr: service.ErrCandidateSessionUnbound},
		{name: "revoked assignment refused", session: session, facts: revoked, now: testdata.Now().Add(time.Minute), wantErr: service.ErrCandidateSessionUnbound},
		{name: "cross assignment facts refused", session: session, facts: other, now: testdata.Now().Add(time.Minute), wantErr: service.ErrCandidateSessionUnbound},
		{name: "missing cookie refused", resolveErr: service.ErrCandidateSessionUnbound, wantErr: service.ErrCandidateSessionUnbound},
	}
	lock := map[string]bool{
		"current session reloads":        true,
		"revoked assignment refused":     true,
		"cross assignment facts refused": true,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := mocks.NewCurrentCandidateSession(t)
			candidates := mocks.NewCandidates(t)
			assignments := mocks.NewAssignmentAccess(t)
			csrf := mocks.NewCSRFTokens(t)
			clock := mocks.NewClock(t)
			if tt.resolveErr != nil {
				current.On("Resolve", mock.Anything).Return(accessport.CandidateSessionRecord{}, tt.resolveErr).Once()
			} else {
				current.On("Resolve", mock.Anything).Return(tt.session, nil).Once()
				clock.On("Now").Return(tt.now)
			}
			if lock[tt.name] {
				assignments.On("WithAssignment", mock.Anything, testdata.AssignmentID, mock.Anything).Return(func(ctx context.Context, assignmentID string, fn func(accessport.AssignmentTx) error) error {
					tx := mocks.NewAssignmentTx(t)
					tx.On("Facts", mock.Anything).Return(tt.facts, nil).Once()
					return fn(tx)
				}).Once()
			}
			if tt.wantErr == nil {
				candidates.On("FindByID", mock.Anything, testdata.CandidateID).Return(testdata.CandidateRecord(t), nil).Once()
				csrf.On("Issue", service.CredentialCandidate, tt.session.ID).Return("candidate-csrf", nil).Once()
			}
			got, err := (accessquery.CandidateSession{
				Current:     current,
				Candidates:  candidates,
				Assignments: assignments,
				CSRF:        csrf,
				Clock:       clock,
			}).Execute(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, testdata.CandidateID, got.Candidate.ID)
			assert.Equal(t, testdata.AssignmentID, got.AssignmentID)
			assert.Equal(t, testdata.CandidateSessionID, got.Session.ID)
			assert.Equal(t, "candidate-csrf", got.CSRFToken)
			assert.Empty(t, got.SessionToken)
		})
	}
}
