package accessquery_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// readCandidateCase names one HR candidate-read outcome.
type readCandidateCase struct {
	name    string
	actor   accessport.AccountRecord
	wantErr error
}

// TestCandidateQueryRequiresCurrentHRAndKnownIdentity hides unknown candidates after authorization.
func TestCandidateQueryRequiresCurrentHRAndKnownIdentity(t *testing.T) {
	admin := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	session := testdata.AdministrativeSessionRecord(t)
	tests := []readCandidateCase{
		{name: "hr reads known candidate", actor: hr},
		{name: "admin cannot read candidate", actor: admin, wantErr: service.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := mocks.NewCandidates(t)
			current := mocks.NewCurrentAdministrativeSession(t)
			clock := mocks.NewClock(t)
			actorSession := session
			actorSession.AccountID = tt.actor.ID
			current.On("Resolve", mock.Anything).Return(actorSession, tt.actor, nil).Once()
			clock.On("Now").Return(testdata.Now())
			if tt.wantErr == nil {
				candidates.On("FindByID", mock.Anything, testdata.CandidateID).Return(testdata.CandidateRecord(t), nil).Once()
			}
			got, err := (accessquery.Candidate{
				Candidates: candidates,
				Current:    current,
				Clock:      clock,
			}).Execute(context.Background(), testdata.CandidateID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, testdata.CandidateID, got.ID)
			assert.Equal(t, "Andi Pratama", got.Name)
			assert.Equal(t, "andi@example.com", got.Email)
		})
	}
}
