package candidateprogress_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/candidateprogress"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// executionID is the execution the fixtures read.
const executionID = "22222222-2222-4222-8222-222222222222"

// at is the deterministic instant of the fixtures.
func at() time.Time { return time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC) }

// baseRecord builds a fresh progress record with one open session that has provider identity and readiness.
func baseRecord(t *testing.T) interviewport.ProgressRecord {
	t.Helper()
	config, err := valueobject.NewHRConfiguration(valueobject.HRInput{
		MandatoryQuestions: []valueobject.QuestionInput{{ID: "background", Stage: "background", Text: "Experience?"}, {ID: "screening", Stage: "screening", Text: "Project?"}, {ID: "compensation", Stage: "compensation", Text: "Expectations?"}},
		StageSeconds:       &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{DiscussionMode: "expectations_only"},
	})
	require.NoError(t, err)
	version, err := agentconfig.NewDraftVersion(1)
	require.NoError(t, err)
	revision, err := agentconfig.RestoreRevision(agentconfig.RevisionSnapshot[valueobject.HRConfiguration]{
		Identity:     agentconfig.RevisionIdentity{ID: "revision", InstanceID: "instance", Number: 1, DraftVersion: version, Template: agentconfig.TemplateRef{Template: "hr", Version: "1", Behavior: "1"}},
		ValidationID: "validation", Configuration: config, CreatedAt: at(),
	})
	require.NoError(t, err)
	p, err := interview.NewProgress(executionID, revision, at())
	require.NoError(t, err)
	provider, ready := "provider-1", at()
	return interviewport.ProgressRecord{
		Snapshot: p.Snapshot(), Version: 3,
		Sessions: []interviewport.VoiceSession{{ID: "session-1", AttemptID: "attempt-1", Stage: enum.StageBackground, ProviderSessionID: &provider, ReadinessAt: &ready, OpenedAt: at()}},
	}
}

// TestReaderCurrent covers the candidate-safe projection, absence and store failure.
func TestReaderCurrent(t *testing.T) {
	storeErr := errors.New("store down")
	// testCase is one progress scenario with its stored record and expected summary.
	type testCase struct {
		name    string
		arrange func(*testing.T, *interviewport.ProgressRecord)
		find    error
		want    *service.CandidateProgress
		wantErr error
	}
	tests := []testCase{
		{
			name:    "ready session in the first stage is connected",
			arrange: func(*testing.T, *interviewport.ProgressRecord) {},
			want:    &service.CandidateProgress{Phase: "Background", Connection: "connected"},
		},
		{
			name:    "screening stage uses its own label",
			arrange: func(_ *testing.T, r *interviewport.ProgressRecord) { r.Snapshot.CurrentStage = enum.StageScreening },
			want:    &service.CandidateProgress{Phase: "Screening", Connection: "connected"},
		},
		{
			name: "active reconnect episode is reconnecting",
			arrange: func(_ *testing.T, r *interviewport.ProgressRecord) {
				detected := at()
				r.Snapshot.Reconnect = interview.ReconnectState{DetectedAt: &detected, Episodes: 1}
			},
			want: &service.CandidateProgress{Phase: "Background", Connection: "reconnecting"},
		},
		{
			name:    "closing stage is finalizing",
			arrange: func(_ *testing.T, r *interviewport.ProgressRecord) { r.Snapshot.CurrentStage = enum.StageClosing },
			want:    &service.CandidateProgress{Phase: "Closing", Connection: "finalizing"},
		},
		{
			name: "session without readiness is not yet progress",
			arrange: func(_ *testing.T, r *interviewport.ProgressRecord) {
				r.Sessions[0].ReadinessAt = nil
			},
		},
		{
			name:    "absent progress is no summary",
			arrange: func(*testing.T, *interviewport.ProgressRecord) {},
			find:    interviewport.ErrNotFound,
		},
		{
			name:    "store failure is returned",
			arrange: func(*testing.T, *interviewport.ProgressRecord) {},
			find:    storeErr,
			wantErr: storeErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := baseRecord(t)
			tt.arrange(t, &record)
			store := mocks.NewProgressStore(t)
			store.On("Find", mock.Anything, executionID).Return(record, tt.find)
			got, err := candidateprogress.Reader{Store: store}.Current(context.Background(), executionID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
