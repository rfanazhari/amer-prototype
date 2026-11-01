package executionrepository_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/executionrepository"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	executionID  = "ffffffff-ffff-4fff-8fff-fffffffffff1"
	assignmentID = "dddddddd-dddd-4ddd-8ddd-ddddddddddd1"
	instanceID   = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8"
	revisionID   = "cccccccc-cccc-4ccc-8ccc-ccccccccccc8"
	progressID   = "99999999-9999-4999-8999-999999999991"
	actor        = "runtime:test"
)

// storeCase names one execution persistence behavior over sqlmock.
type storeCase struct {
	name string
	run  func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock)
}

// executionColumns lists the agent_executions columns selected on reads.
var executionColumns = []string{
	"id", "instance_id", "published_revision_id", "state", "state_version", "committed_progress_ref",
	"activated_at", "terminal_at", "terminal_reason", "created_at", "updated_at",
}

// createdAt is the controlled admission instant used by every case.
var createdAt = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

// activatedAt is the controlled activation instant used by every case.
var activatedAt = createdAt.Add(time.Minute)

// terminalAt is the controlled terminal instant used by every case.
var terminalAt = createdAt.Add(2 * time.Minute)

// newCreated builds a CREATED execution snapshot and its initial transition through the domain factory.
func newCreated(t *testing.T) (execution.Snapshot, execution.TransitionFact) {
	t.Helper()
	_, snap, fact, err := execution.NewExecution(executionID, execution.Binding{InstanceID: instanceID, RevisionID: revisionID}, createdAt)
	require.NoError(t, err)
	fact.ActorRef = actor
	return snap, fact
}

// runningSnapshot restores a RUNNING execution through the domain factory and activation.
func runningSnapshot(t *testing.T) execution.Snapshot {
	t.Helper()
	snap, _ := newCreated(t)
	agg, err := execution.RestoreExecution(snap)
	require.NoError(t, err)
	running, _, err := agg.Activate(1, actor, "op-1", 1, nil, activatedAt)
	require.NoError(t, err)
	return running
}

// failedSnapshot restores a FAILED execution through the domain factory.
func failedSnapshot(t *testing.T) execution.Snapshot {
	t.Helper()
	agg, err := execution.RestoreExecution(runningSnapshot(t))
	require.NoError(t, err)
	failed, _, err := agg.Fail("provider_lost", 2, 1, nil, actor, "op-1", terminalAt)
	require.NoError(t, err)
	return failed
}

// rowFor returns the sqlmock row that stores snap in agent_executions.
func rowFor(snap execution.Snapshot) *sqlmock.Rows {
	var progress, reason any
	if snap.ProgressRef != nil {
		progress = string(*snap.ProgressRef)
	}
	if snap.TerminalReason != "" {
		reason = snap.TerminalReason
	}
	var activated, terminal any
	if snap.ActivatedAt != nil {
		activated = *snap.ActivatedAt
	}
	if snap.TerminalAt != nil {
		terminal = *snap.TerminalAt
	}
	return sqlmock.NewRows(executionColumns).AddRow(
		snap.ID, snap.Binding.InstanceID, snap.Binding.RevisionID, string(snap.State), snap.StateVersion,
		progress, activated, terminal, reason, snap.CreatedAt, snap.UpdatedAt,
	)
}

// sqlStore opens a GORM store over sqlmock whose matcher rejects binding columns in agent_executions updates.
func sqlStore(t *testing.T) (executionrepository.Store, sqlmock.Sqlmock) {
	t.Helper()
	matcher := sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		if strings.HasPrefix(actual, `UPDATE "agent_executions"`) &&
			(strings.Contains(actual, "instance_id") || strings.Contains(actual, "published_revision_id")) {
			return errors.New("agent_executions update must omit the binding columns: " + actual)
		}
		if !regexp.MustCompile("(?is)" + expected).MatchString(actual) {
			return errors.New("query does not match " + expected + ": " + actual)
		}
		return nil
	})
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)
	return executionrepository.Store{DB: gdb}, mock
}

// expectCreatedInserts expects the execution row and then its initial transition row.
func expectCreatedInserts(mock sqlmock.Sqlmock, snap execution.Snapshot) {
	mock.ExpectExec(`INSERT INTO "agent_executions"`).WithArgs(
		snap.ID, snap.Binding.InstanceID, snap.Binding.RevisionID, "CREATED", int64(1), nil, nil, nil, nil, snap.CreatedAt, snap.UpdatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO "agent_execution_transitions"`).WithArgs(
		sqlmock.AnyArg(), snap.ID, nil, "CREATED", int64(1), actor, nil, nil, snap.CreatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))
}

// TestExecutionStoreInsert covers admission writes, their order and their classified failures.
func TestExecutionStoreInsert(t *testing.T) {
	tests := []storeCase{
		{
			name: "interview insert writes the execution before the assignment pin",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				snap, fact := newCreated(t)
				mock.ExpectBegin()
				expectCreatedInserts(mock, snap)
				mock.ExpectExec(`INSERT INTO "interview_executions"`).WithArgs(executionID, assignmentID, instanceID, revisionID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				require.NoError(t, store.InsertInterview(context.Background(), assignmentID, snap, fact))
			},
		},
		{
			name: "admit insert writes no pin",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				snap, fact := newCreated(t)
				mock.ExpectBegin()
				expectCreatedInserts(mock, snap)
				mock.ExpectCommit()
				require.NoError(t, store.Insert(context.Background(), snap, fact))
			},
		},
		{
			name: "duplicate assignment pin is reported as already existing and rolled back",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				snap, fact := newCreated(t)
				mock.ExpectBegin()
				expectCreatedInserts(mock, snap)
				mock.ExpectExec(`INSERT INTO "interview_executions"`).WillReturnError(&pgconn.PgError{Code: "23505"})
				mock.ExpectRollback()
				err := store.InsertInterview(context.Background(), assignmentID, snap, fact)
				assert.ErrorIs(t, err, executionport.ErrAlreadyExists)
			},
		},
		{
			name: "missing revision is a dependency failure and rolled back",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				snap, fact := newCreated(t)
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO "agent_executions"`).WillReturnError(&pgconn.PgError{Code: "23503"})
				mock.ExpectRollback()
				err := store.Insert(context.Background(), snap, fact)
				assert.ErrorIs(t, err, executionport.ErrDependencyUnavailable)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}

// TestExecutionStoreUpdate covers the execution row lock, explicit column lists and first-terminal-wins updates.
func TestExecutionStoreUpdate(t *testing.T) {
	lock := `SELECT .* FROM "agent_executions" WHERE id = \$1.* FOR UPDATE`
	tests := []storeCase{
		{
			name: "completion updates owned columns explicitly and appends the transition",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				running := runningSnapshot(t)
				mock.ExpectBegin()
				mock.ExpectQuery(lock).WithArgs(executionID, 1).WillReturnRows(rowFor(running))
				// GORM sorts map keys, so the argument order is alphabetical by column.
				mock.ExpectExec(`UPDATE "agent_executions" SET "activated_at"=.*"committed_progress_ref"=.*"state"=.*"state_version"=.*"terminal_at"=.*"terminal_reason"=.*"updated_at"=.* WHERE .*state IN`).
					WithArgs(activatedAt, progressID, "COMPLETED", int64(3), terminalAt, nil, terminalAt, executionID, int64(2), "CREATED", "RUNNING").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO "agent_execution_transitions"`).WithArgs(
					sqlmock.AnyArg(), executionID, "RUNNING", "COMPLETED", int64(3), actor, "op-1", nil, terminalAt,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				err := store.Update(context.Background(), executionID, func(s *execution.Snapshot) (*execution.TransitionFact, error) {
					agg, err := execution.RestoreExecution(*s)
					require.NoError(t, err)
					next, fact, err := agg.Complete(execution.ProgressRef(progressID), 2, 1, nil, actor, "op-1", terminalAt)
					require.NoError(t, err)
					*s = next
					return &fact, nil
				})
				require.NoError(t, err)
			},
		},
		{
			name: "progress commit writes null terminal facts and records no transition",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				running := runningSnapshot(t)
				mock.ExpectBegin()
				mock.ExpectQuery(lock).WithArgs(executionID, 1).WillReturnRows(rowFor(running))
				mock.ExpectExec(`UPDATE "agent_executions" SET`).
					WithArgs(activatedAt, progressID, "RUNNING", int64(2), nil, nil, terminalAt, executionID, int64(2), "CREATED", "RUNNING").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				err := store.Update(context.Background(), executionID, func(s *execution.Snapshot) (*execution.TransitionFact, error) {
					agg, err := execution.RestoreExecution(*s)
					require.NoError(t, err)
					next, err := agg.CommitProgress(execution.ProgressRef(progressID), 2, 1, nil, terminalAt)
					require.NoError(t, err)
					*s = next
					return nil, nil
				})
				require.NoError(t, err)
			},
		},
		{
			name: "changed terminal row updates zero rows and reports the terminal conflict",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				failed := failedSnapshot(t)
				mock.ExpectBegin()
				mock.ExpectQuery(lock).WithArgs(executionID, 1).WillReturnRows(rowFor(failed))
				mock.ExpectExec(`UPDATE "agent_executions" SET`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
				err := store.Update(context.Background(), executionID, func(s *execution.Snapshot) (*execution.TransitionFact, error) {
					s.State = execution.StateCancelled
					s.StateVersion++
					return nil, nil
				})
				assert.ErrorIs(t, err, execution.ErrTerminalConflict)
			},
		},
		{
			name: "zero rows on a nonterminal row reports a stale version",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				running := runningSnapshot(t)
				mock.ExpectBegin()
				mock.ExpectQuery(lock).WithArgs(executionID, 1).WillReturnRows(rowFor(running))
				mock.ExpectExec(`UPDATE "agent_executions" SET`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
				err := store.Update(context.Background(), executionID, func(s *execution.Snapshot) (*execution.TransitionFact, error) {
					s.UpdatedAt = terminalAt
					return nil, nil
				})
				assert.ErrorIs(t, err, execution.ErrStaleVersion)
			},
		},
		{
			name: "unchanged terminal replay writes nothing",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				failed := failedSnapshot(t)
				mock.ExpectBegin()
				mock.ExpectQuery(lock).WithArgs(executionID, 1).WillReturnRows(rowFor(failed))
				mock.ExpectCommit()
				err := store.Update(context.Background(), executionID, func(s *execution.Snapshot) (*execution.TransitionFact, error) {
					assert.Equal(t, execution.StateFailed, s.State)
					return nil, nil
				})
				require.NoError(t, err)
			},
		},
		{
			name: "callback failure rolls back without updating",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				running := runningSnapshot(t)
				mock.ExpectBegin()
				mock.ExpectQuery(lock).WithArgs(executionID, 1).WillReturnRows(rowFor(running))
				mock.ExpectRollback()
				err := store.Update(context.Background(), executionID, func(*execution.Snapshot) (*execution.TransitionFact, error) {
					return nil, execution.ErrStaleAttempt
				})
				assert.ErrorIs(t, err, execution.ErrStaleAttempt)
			},
		},
		{
			name: "missing execution is not found",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(lock).WithArgs(executionID, 1).WillReturnRows(sqlmock.NewRows(executionColumns))
				mock.ExpectRollback()
				err := store.Update(context.Background(), executionID, func(*execution.Snapshot) (*execution.TransitionFact, error) {
					return nil, nil
				})
				assert.ErrorIs(t, err, executionport.ErrNotFound)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}

// TestExecutionStoreWithAssignment covers the assignment-first lock order and transaction sharing.
func TestExecutionStoreWithAssignment(t *testing.T) {
	assignmentLock := `SELECT id FROM interview_assignments WHERE id = \$1 FOR UPDATE`
	facts := `SELECT id, instance_id, published_revision_id, access_revoked_at, completed_at FROM interview_assignments WHERE id = \$1`
	factColumns := []string{"id", "instance_id", "published_revision_id", "access_revoked_at", "completed_at"}
	tests := []storeCase{
		{
			name: "assignment is locked before the execution inside one transaction",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				running := runningSnapshot(t)
				revokedAt := createdAt.Add(30 * time.Second)
				mock.ExpectBegin()
				mock.ExpectExec(assignmentLock).WithArgs(assignmentID).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery(facts).WithArgs(assignmentID).WillReturnRows(
					sqlmock.NewRows(factColumns).AddRow(assignmentID, instanceID, revisionID, revokedAt, nil))
				mock.ExpectQuery(`SELECT .* FROM "agent_executions" WHERE id = \$1.* FOR UPDATE`).WithArgs(executionID, 1).WillReturnRows(rowFor(running))
				mock.ExpectCommit()
				var got executionport.AssignmentRow
				err := store.WithAssignment(context.Background(), assignmentID, func(ctx context.Context, row executionport.AssignmentRow) error {
					got = row
					return store.Update(ctx, executionID, func(*execution.Snapshot) (*execution.TransitionFact, error) { return nil, nil })
				})
				require.NoError(t, err)
				assert.Equal(t, executionport.AssignmentRow{
					AssignmentID: assignmentID, InstanceID: instanceID, RevisionID: revisionID, AccessRevokedAt: &revokedAt,
				}, got)
			},
		},
		{
			name: "missing assignment is not found and skips the callback",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(assignmentLock).WithArgs(assignmentID).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(facts).WithArgs(assignmentID).WillReturnRows(sqlmock.NewRows(factColumns))
				mock.ExpectRollback()
				called := false
				err := store.WithAssignment(context.Background(), assignmentID, func(context.Context, executionport.AssignmentRow) error {
					called = true
					return nil
				})
				assert.ErrorIs(t, err, executionport.ErrNotFound)
				assert.False(t, called)
			},
		},
		{
			name: "callback failure rolls back the lock transaction",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(assignmentLock).WithArgs(assignmentID).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery(facts).WithArgs(assignmentID).WillReturnRows(
					sqlmock.NewRows(factColumns).AddRow(assignmentID, instanceID, revisionID, nil, nil))
				mock.ExpectRollback()
				err := store.WithAssignment(context.Background(), assignmentID, func(context.Context, executionport.AssignmentRow) error {
					return executionport.ErrAssignmentUnavailable
				})
				assert.ErrorIs(t, err, executionport.ErrAssignmentUnavailable)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}

// TestExecutionStoreFind covers restoration, UTC times, absence and corrupt rows.
func TestExecutionStoreFind(t *testing.T) {
	tests := []storeCase{
		{
			name: "find restores a running execution with UTC times",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				zone := time.FixedZone("WIB", 7*3600)
				mock.ExpectQuery(`SELECT .* FROM "agent_executions" WHERE id = \$1`).WithArgs(executionID, 1).
					WillReturnRows(sqlmock.NewRows(executionColumns).AddRow(
						executionID, instanceID, revisionID, "RUNNING", int64(2), nil, activatedAt.In(zone), nil, nil, createdAt.In(zone), activatedAt.In(zone)))
				got, err := store.Find(context.Background(), executionID)
				require.NoError(t, err)
				assert.Equal(t, execution.Binding{InstanceID: instanceID, RevisionID: revisionID}, got.Binding)
				assert.Equal(t, execution.StateRunning, got.State)
				assert.Equal(t, int64(2), got.StateVersion)
				assert.Equal(t, createdAt, got.CreatedAt)
				require.NotNil(t, got.ActivatedAt)
				assert.Equal(t, activatedAt, *got.ActivatedAt)
				assert.Nil(t, got.TerminalAt)
				assert.Nil(t, got.ProgressRef)
			},
		},
		{
			name: "find reports an absent execution as not found",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "agent_executions" WHERE id = \$1`).WithArgs(executionID, 1).
					WillReturnRows(sqlmock.NewRows(executionColumns))
				_, err := store.Find(context.Background(), executionID)
				assert.ErrorIs(t, err, executionport.ErrNotFound)
			},
		},
		{
			name: "find classifies an unknown stored state as technical",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "agent_executions" WHERE id = \$1`).WithArgs(executionID, 1).
					WillReturnRows(sqlmock.NewRows(executionColumns).AddRow(
						executionID, instanceID, revisionID, "WAITING_INPUT", int64(1), nil, nil, nil, nil, createdAt, createdAt))
				_, err := store.Find(context.Background(), executionID)
				var technical errs.Technical
				assert.ErrorAs(t, err, &technical)
				assert.ErrorIs(t, err, execution.ErrInvalidExecution)
			},
		},
		{
			name: "find by assignment joins the interview pin without inserting",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				snap, _ := newCreated(t)
				mock.ExpectQuery(`SELECT .*"agent_executions".* FROM "agent_executions" JOIN interview_executions .*assignment_id = \$1`).
					WithArgs(assignmentID, 1).WillReturnRows(rowFor(snap))
				got, err := store.FindByAssignment(context.Background(), assignmentID)
				require.NoError(t, err)
				assert.Equal(t, snap, got)
			},
		},
		{
			name: "find by assignment reports a missing pin as not found",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM "agent_executions" JOIN interview_executions`).WithArgs(assignmentID, 1).
					WillReturnRows(sqlmock.NewRows(executionColumns))
				_, err := store.FindByAssignment(context.Background(), assignmentID)
				assert.ErrorIs(t, err, executionport.ErrNotFound)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}

// TestExecutionStoreWithTransaction covers the shared admit transaction: joined writes, commit and rollback.
func TestExecutionStoreWithTransaction(t *testing.T) {
	tests := []storeCase{
		{
			name: "writes made with the handed context share one transaction",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				snap, fact := newCreated(t)
				mock.ExpectBegin()
				expectCreatedInserts(mock, snap)
				mock.ExpectCommit()
				err := store.WithTransaction(context.Background(), func(ctx context.Context) error {
					return store.Insert(ctx, snap, fact)
				})
				require.NoError(t, err)
			},
		},
		{
			name: "callback failure rolls back and returns the callback error",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				snap, fact := newCreated(t)
				boom := errors.New("command conflict")
				mock.ExpectBegin()
				expectCreatedInserts(mock, snap)
				mock.ExpectRollback()
				err := store.WithTransaction(context.Background(), func(ctx context.Context) error {
					require.NoError(t, store.Insert(ctx, snap, fact))
					return boom
				})
				require.ErrorIs(t, err, boom)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}
