package accessusecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// pruneCall is one scripted DeleteStale outcome.
type pruneCall struct {
	removed int64
	err     error
}

// pruneCase names one bounded cleanup run.
type pruneCase struct {
	name      string
	calls     []pruneCall
	cancelled bool
	wantTotal int64
	wantErr   error
}

// TestPruneSessionsRemovesStaleRowsInBoundedBatches covers the cutoff, batch continuation, interruption and cancellation.
func TestPruneSessionsRemovesStaleRowsInBoundedBatches(t *testing.T) {
	now := testdata.Now().Add(30 * 24 * time.Hour)
	cutoff := now.Add(-7 * 24 * time.Hour)
	tests := []pruneCase{
		{name: "nothing stale", calls: []pruneCall{{removed: 0}}, wantTotal: 0},
		{name: "partial batch stops", calls: []pruneCall{{removed: 120}}, wantTotal: 120},
		{name: "full batches continue until a short one", calls: []pruneCall{{removed: 500}, {removed: 500}, {removed: 7}}, wantTotal: 1007},
		{name: "exactly full final batch is confirmed empty", calls: []pruneCall{{removed: 500}, {removed: 0}}, wantTotal: 500},
		{name: "failure keeps removed count and is returned", calls: []pruneCall{{removed: 500}, {err: assert.AnError}}, wantTotal: 500, wantErr: assert.AnError},
		{name: "cancelled run deletes nothing", cancelled: true, wantTotal: 0, wantErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retention := mocks.NewSessionRetention(t)
			clock := mocks.NewClock(t)
			clock.On("Now").Return(now.Add(321 * time.Microsecond)).Maybe()
			for _, call := range tt.calls {
				retention.On("DeleteStale", mock.Anything, mock.MatchedBy(func(got time.Time) bool { return got.Equal(cutoff) }), 500).Return(call.removed, call.err).Once()
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancelled {
				cancel()
			}
			total, err := (accessusecase.PruneSessions{Retention: retention, Clock: clock}).Execute(ctx)
			assert.Equal(t, tt.wantTotal, total)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
