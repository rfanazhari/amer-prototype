package agentbindingrepository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentbindingusecase"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentbindingrepository"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// createAssignmentSQLCase names one create-path SQL pin sequence.
type createAssignmentSQLCase struct {
	name string
	run  func(t *testing.T, sqlMock sqlmock.Sqlmock, command agentbindingusecase.CreateAssignment)
}

// TestCreateAssignmentPersistenceSQL covers candidate lookup then assignment pin insert with sqlmock.
func TestCreateAssignmentPersistenceSQL(t *testing.T) {
	candidate := testdata.CandidateRecord(t)
	tests := []createAssignmentSQLCase{
		{
			name: "create selects candidate then inserts published revision pin",
			run: func(t *testing.T, sqlMock sqlmock.Sqlmock, command agentbindingusecase.CreateAssignment) {
				rows := sqlmock.NewRows([]string{"id", "name", "email", "created_by", "created_at", "updated_at"}).
					AddRow(candidate.ID, candidate.Name, candidate.Email, candidate.CreatedBy, candidate.CreatedAt, candidate.UpdatedAt)
				sqlMock.ExpectQuery(`(?i)SELECT .* FROM ["']?candidates["']?`).WithArgs(candidate.ID, 1).WillReturnRows(rows)
				sqlMock.ExpectBegin()
				sqlMock.ExpectExec(`(?i)INSERT INTO ["']?interview_assignments["']?`).WithArgs(
					sqlmock.AnyArg(), candidate.ID, instanceID, revisionID,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				sqlMock.ExpectCommit()
				got, err := command.Execute(context.Background(), candidate.ID, instanceID)
				require.NoError(t, err)
				require.NotNil(t, got)
				require.NotEmpty(t, got.ID)
				require.Equal(t, revisionID, got.PublishedRevisionID)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, sqlMock := sqlCreateAssignment(t)
			tt.run(t, sqlMock, command)
		})
	}
}

// sqlCreateAssignment opens candidate and assignment adapters over sqlmock with a mocked instance reader.
func sqlCreateAssignment(t *testing.T) (agentbindingusecase.CreateAssignment, sqlmock.Sqlmock) {
	t.Helper()
	db, sqlMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlMock.ExpectationsWereMet())
		_ = db.Close()
	})
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	session := testdata.AdministrativeSessionRecord(t)
	session.AccountID = hr.ID
	current := mocks.NewCurrentAdministrativeSession(t)
	clock := mocks.NewClock(t)
	reader := mocks.NewReader[agentconfigport.StoredInstance, agentconfigport.StoredRevision](t)
	current.On("Resolve", mock.Anything).Return(session, hr, nil).Maybe()
	clock.On("Now").Return(testdata.Now()).Maybe()
	reader.On("FindInstance", mock.Anything, instanceID).Return(storedHRForSQL(t), nil).Maybe()
	return agentbindingusecase.CreateAssignment{
		Current:     current,
		Clock:       clock,
		Candidates:  accessrepository.CandidateIdentities{Store: accessrepository.Store{DB: gdb}},
		Instances:   reader,
		Assignments: agentbindingrepository.Assignments{DB: gdb},
	}, sqlMock
}

// storedHRForSQL restores an HR Interviewer instance whose active pointer is revisionID.
func storedHRForSQL(t *testing.T) agentconfigport.StoredInstance {
	t.Helper()
	cfg, err := valueobject.NewHRConfiguration(valueobject.HRInput{})
	require.NoError(t, err)
	template := aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1}
	version, err := aggregate.NewDraftVersion(1)
	require.NoError(t, err)
	active := &aggregate.RevisionIdentity{ID: revisionID, InstanceID: instanceID, Number: 1, DraftVersion: version, Template: template}
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
		ID: instanceID, Template: template, DraftVersion: version, Configuration: cfg,
		Active: active, CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: "Backend interviewer", HR: instance}
}
