package executionusecase_test

import (
	"context"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestRecordAttempt verifies durable authority, exact budget boundaries and rejected writes.
func TestRecordAttempt(t *testing.T) {
	// testCase describes a bounded generic admission.
	type testCase struct {
		name     string
		state    execution.State
		version  int64
		previous int
		expired  bool
		wantErr  bool
	}
	tests := []testCase{
		{name: "first admission activates", state: execution.StateCreated, version: 1},
		{name: "replacement increments authority", state: execution.StateRunning, version: 2, previous: 1},
		{name: "stale version writes nothing", state: execution.StateRunning, version: 1, wantErr: true},
		{name: "terminal writes nothing", state: execution.StateCancelled, version: 2, wantErr: true},
		{name: "third admission durably fails", state: execution.StateRunning, version: 2, previous: 2, wantErr: true},
		{name: "window equality durably fails", state: execution.StateRunning, version: 2, previous: 1, expired: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLifecycleFixture(t, snapshotIn(t, tt.state), nil)
			ledger := mocks.NewAttemptLedger(t)
			history := []executionport.Attempt{}
			for i := 0; i < tt.previous; i++ {
				history = append(history, executionport.Attempt{ID: "previous", ExecutionID: existingID, OperationKey: opKey, Generation: int64(i + 1), Number: i + 1, Outcome: "FAILED", CreatedAt: trustedNow().Add(-time.Minute), Deadline: trustedNow().Add(-time.Second)})
			}
			ledger.On("List", mock.Anything, existingID, opKey).Return(history, nil).Maybe()
			var inserted *executionport.Attempt
			ledger.On("Insert", mock.Anything, mock.Anything).Return(func(_ context.Context, a executionport.Attempt) error { inserted = &a; return nil }).Maybe()
			in := executionusecase.RecordAttemptInput{ExecutionID: existingID, AttemptID: existingID, ExpectedVersion: tt.version, OperationKey: opKey, RequestKey: "request", ActorRef: runtimeActor, Generation: 1, Limit: 2, Window: 90 * time.Second, AttemptTimeout: 30 * time.Second}
			if tt.expired {
				in.WindowStart = trustedNow().Add(-90 * time.Second)
			}
			op := executionusecase.RecordAttempt{Transactions: f.tx, Executions: f.store, Attempts: ledger, Clock: testClock(t)}
			got, err := op.Execute(context.Background(), in)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, inserted)
				if tt.previous == 2 || tt.expired {
					require.ErrorIs(t, err, executionport.ErrAttemptBudget)
					require.Equal(t, execution.StateFailed, f.after.State)
				} else {
					f.requireNothingWritten(t)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, inserted)
			require.Equal(t, execution.StateRunning, f.after.State)
			require.Equal(t, int64(tt.previous+1), inserted.Generation)
			require.Equal(t, tt.version+1, got.StateVersion)
			require.Equal(t, inserted.Generation, f.after.CurrentGeneration)
		})
	}
}
