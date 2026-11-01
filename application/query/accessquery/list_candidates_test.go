package accessquery_test

import (
	"context"
	"testing"
	"time"

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

const otherCreatorID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb9"

// listCandidatesCase names one current-HR candidate list outcome.
type listCandidatesCase struct {
	name       string
	actor      accessport.AccountRecord
	records    []accessport.CandidateRecord
	wantErr    error
	wantIDs    []string
	forbidRead bool
}

// TestListCandidatesReturnsOnlyTheCurrentHRCreator keeps another creator's rows out and refuses ADMIN before any read.
func TestListCandidatesReturnsOnlyTheCurrentHRCreator(t *testing.T) {
	admin := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	own := testdata.CandidateRecord(t)
	own.CreatedBy = hr.ID
	newer := testdata.CandidateRecord(t)
	newer.ID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2"
	newer.Name = "Budi Santoso"
	newer.Email = "budi@example.com"
	newer.CreatedBy = hr.ID
	newer.CreatedAt = testdata.Now().Add(time.Minute)
	newer.UpdatedAt = newer.CreatedAt
	other := testdata.CandidateRecord(t)
	other.ID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa3"
	other.Name = "Citra Lestari"
	other.Email = "citra@example.com"
	other.CreatedBy = otherCreatorID
	session := testdata.AdministrativeSessionRecord(t)
	tests := []listCandidatesCase{
		{
			name:    "own rows only",
			actor:   hr,
			records: []accessport.CandidateRecord{newer, own},
			wantIDs: []string{newer.ID, own.ID},
		},
		{
			name:    "other creator excluded",
			actor:   hr,
			records: []accessport.CandidateRecord{own},
			wantIDs: []string{own.ID},
		},
		{
			name:       "admin forbidden before read",
			actor:      admin,
			wantErr:    service.ErrForbidden,
			forbidRead: true,
		},
		{
			name:    "empty list",
			actor:   hr,
			records: []accessport.CandidateRecord{},
			wantIDs: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := mocks.NewCandidates(t)
			current := mocks.NewCurrentAdministrativeSession(t)
			clock := mocks.NewClock(t)
			actorSession := session
			actorSession.AccountID = tt.actor.ID
			current.On("Resolve", mock.Anything).Return(actorSession, tt.actor, nil).Once()
			clock.On("Now").Return(testdata.Now()).Once()
			if !tt.forbidRead {
				candidates.On("ListByCreator", mock.Anything, tt.actor.ID).Return(tt.records, nil).Once()
			}
			got, err := (accessquery.ListCandidates{
				Candidates: candidates,
				Current:    current,
				Clock:      clock,
			}).Execute(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				candidates.AssertNotCalled(t, "ListByCreator", mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Len(t, got, len(tt.wantIDs))
			for i, id := range tt.wantIDs {
				assert.Equal(t, id, got[i].ID)
				assert.NotEqual(t, other.ID, got[i].ID)
				assert.NotEqual(t, otherCreatorID, tt.actor.ID)
			}
			if tt.name == "own rows only" {
				assert.Equal(t, newer.Name, got[0].Name)
				assert.Equal(t, newer.Email, got[0].Email)
				assert.True(t, newer.CreatedAt.Equal(got[0].CreatedAt))
				assert.True(t, newer.UpdatedAt.Equal(got[0].UpdatedAt))
			}
			candidates.AssertNotCalled(t, "ListByCreator", mock.Anything, otherCreatorID)
		})
	}
}
