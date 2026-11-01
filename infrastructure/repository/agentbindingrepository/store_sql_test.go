package agentbindingrepository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentbindingrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	instanceID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8"
	revisionID = "cccccccc-cccc-4ccc-8ccc-ccccccccccc8"
)

// bindingSQLCase names one assignment or execution SQL pin behavior.
type bindingSQLCase struct {
	name string
	run  func(t *testing.T, assignments agentbindingrepository.Assignments, executions agentbindingrepository.Executions, mock sqlmock.Sqlmock)
}

// TestBindingPersistenceSQL covers Interview-shaped pin inserts and lookups with sqlmock.
func TestBindingPersistenceSQL(t *testing.T) {
	assignment := agentbindingport.AssignmentBinding{
		AssignmentID: testdata.AssignmentID, CandidateID: testdata.CandidateID, InstanceID: instanceID, RevisionID: revisionID,
	}
	execution := agentbindingport.ExecutionBinding{
		ExecutionID: testdata.ExecutionID, AssignmentID: testdata.AssignmentID, InstanceID: instanceID, RevisionID: revisionID,
	}
	tests := []bindingSQLCase{
		{
			name: "assignment insert writes published revision pin",
			run: func(t *testing.T, assignments agentbindingrepository.Assignments, _ agentbindingrepository.Executions, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)INSERT INTO ["']?interview_assignments["']?`).WithArgs(
					assignment.AssignmentID, assignment.CandidateID, assignment.InstanceID, assignment.RevisionID,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				require.NoError(t, assignments.Insert(context.Background(), assignment))
			},
		},
		{
			name: "assignment find restores pinned revision",
			run: func(t *testing.T, assignments agentbindingrepository.Assignments, _ agentbindingrepository.Executions, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_assignments["']?`).WithArgs(testdata.AssignmentID, 1).WillReturnRows(
					sqlmock.NewRows([]string{"id", "candidate_id", "instance_id", "published_revision_id"}).
						AddRow(assignment.AssignmentID, assignment.CandidateID, assignment.InstanceID, assignment.RevisionID),
				)
				got, err := assignments.Find(context.Background(), testdata.AssignmentID)
				require.NoError(t, err)
				require.Equal(t, assignment, got)
			},
		},
		{
			name: "execution insert writes assignment pin at start",
			run: func(t *testing.T, _ agentbindingrepository.Assignments, executions agentbindingrepository.Executions, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)INSERT INTO ["']?interview_executions["']?`).WithArgs(
					execution.ExecutionID, execution.AssignmentID, execution.InstanceID, execution.RevisionID,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				require.NoError(t, executions.Insert(context.Background(), execution))
			},
		},
		{
			name: "execution find restores pinned revision after later publication",
			run: func(t *testing.T, _ agentbindingrepository.Assignments, executions agentbindingrepository.Executions, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_executions["']?`).WithArgs(testdata.ExecutionID, 1).WillReturnRows(
					sqlmock.NewRows([]string{"id", "assignment_id", "instance_id", "published_revision_id"}).
						AddRow(execution.ExecutionID, execution.AssignmentID, execution.InstanceID, execution.RevisionID),
				)
				got, err := executions.Find(context.Background(), testdata.ExecutionID)
				require.NoError(t, err)
				require.Equal(t, execution, got)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignments, executions, mock := sqlStores(t)
			tt.run(t, assignments, executions, mock)
		})
	}
}

// sqlStores opens GORM adapters over sqlmock with ping disabled so tests own SQL expectations.
func sqlStores(t *testing.T) (agentbindingrepository.Assignments, agentbindingrepository.Executions, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)
	return agentbindingrepository.Assignments{DB: gdb}, agentbindingrepository.Executions{DB: gdb}, mock
}
