package executionusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestFencedWrite checks that rejected authority never invokes the sibling write callback.
func TestFencedWrite(t *testing.T) {
	// testCase selects a stored authority and deadline boundary.
	type testCase struct {
		name        string
		state       execution.State
		version     int64
		generation  int64
		maximum     time.Duration
		callbackErr bool
		wantErr     bool
	}
	tests := []testCase{
		{name: "current authority invokes callback", state: execution.StateRunning, version: 2, generation: 1},
		{name: "replaced authority refuses callback", state: execution.StateRunning, version: 2, generation: 0, wantErr: true},
		{name: "stale version refuses callback", state: execution.StateRunning, version: 1, generation: 1, wantErr: true},
		{name: "terminal refuses callback", state: execution.StateCancelled, version: 2, generation: 1, wantErr: true},
		{name: "expired hard deadline refuses callback", state: execution.StateRunning, version: 2, generation: 1, maximum: 15 * time.Minute, wantErr: true},
		{name: "callback error propagates", state: execution.StateRunning, version: 2, generation: 1, callbackErr: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLifecycleFixture(t, snapshotIn(t, tt.state), nil)
			ledger := mocks.NewAttemptLedger(t)
			ledger.On("Find", mock.Anything, existingID, "attempt").Return(executionport.Attempt{Generation: tt.generation}, nil).Maybe()
			op := executionusecase.FencedWrite{Transactions: f.tx, Executions: f.store, Attempts: ledger, Clock: testClock(t)}
			called := false
			failure := errors.New("callback failure")
			err := op.Within(context.Background(), existingID, "attempt", tt.version, tt.maximum, func(context.Context) error {
				called = true
				if tt.callbackErr {
					return failure
				}
				return nil
			})
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.callbackErr, called)
			} else {
				require.NoError(t, err)
				require.True(t, called)
			}
			f.requireNothingWritten(t)
		})
	}
}
