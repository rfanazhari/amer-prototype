package execution_test

import (
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestExecutionRecordAttempt covers monotonic replacement and preservation of trusted activation.
func TestExecutionRecordAttempt(t *testing.T) {
	// testCase supplies explicit version and generation expectations.
	type testCase struct {
		name       string
		version    int64
		generation int64
		wantErr    error
	}
	tests := []testCase{
		{name: "replace current authority", version: 2, generation: 2},
		{name: "same generation refused", version: 2, generation: 1, wantErr: execution.ErrStaleAttempt},
		{name: "lower generation refused", version: 2, generation: 0, wantErr: execution.ErrStaleAttempt},
		{name: "stale version refused", version: 1, generation: 2, wantErr: execution.ErrStaleVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := fixtureRunning(t)
			before := e.Snapshot()
			got, err := e.RecordAttempt(tt.version, tt.generation, fixtureNow().Add(time.Second))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, before, e.Snapshot())
				return
			}
			require.NoError(t, err)
			require.Equal(t, int64(3), got.StateVersion)
			require.Equal(t, int64(2), got.CurrentGeneration)
			require.Equal(t, before.ActivatedAt, got.ActivatedAt)
		})
	}
}
