package accessusecase_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// createCandidateCase names one HR candidate-creation outcome.
type createCandidateCase struct {
	name      string
	actor     accessport.AccountRecord
	wantErr   error
	wantWrite bool
}

// TestCreateCandidateRequiresCurrentHRAndDoesNotDedupEmail creates a new identity for every accepted contact.
func TestCreateCandidateRequiresCurrentHRAndDoesNotDedupEmail(t *testing.T) {
	admin := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	session := testdata.AdministrativeSessionRecord(t)
	tests := []createCandidateCase{
		{name: "hr creates candidate", actor: hr, wantWrite: true},
		{name: "admin cannot create candidate", actor: admin, wantErr: service.ErrForbidden},
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
			if tt.wantWrite {
				candidates.On("Insert", mock.Anything, mock.MatchedBy(func(record accessport.CandidateRecord) bool {
					return record.Name == "Andi Pratama" &&
						record.Email == "andi@example.com" &&
						record.CreatedBy == hr.ID
				})).Return(nil).Once()
			}
			got, err := (accessusecase.CreateCandidate{
				Candidates: candidates,
				Current:    current,
				Clock:      clock,
			}).Execute(context.Background(), " Andi Pratama ", "Andi@Example.com")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "Andi Pratama", got.Name)
			assert.Equal(t, "andi@example.com", got.Email)
			assert.NotEmpty(t, got.ID)
		})
	}
}
