package interviewrepository_test

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewrepository"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	execID     = "ffffffff-ffff-4fff-8fff-fffffffffff1"
	sessionOne = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1"
	sessionTwo = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2"
	attemptOne = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1"
	attemptTwo = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb2"
	entryOne   = "cccccccc-cccc-4ccc-8ccc-ccccccccccc1"
	revisionID = "pinned-revision"
)

// base is the controlled RUNNING instant; every scenario time is an offset from it.
var base = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

// now is the trusted application instant returned by the mocked clock.
var now = base.Add(5 * time.Minute)

// progressColumns, coverageColumns and the following lists name the version 10 columns scanned on reads.
var (
	progressColumns   = []string{"execution_id", "follow_up_limit", "current_stage", "stage_activated_at", "committed_version", "controller_id", "created_at", "updated_at"}
	coverageColumns   = []string{"id", "execution_id", "question_id", "stage", "state", "omission_reason", "clarification_count", "answer_entry_id", "updated_at"}
	sessionColumns    = []string{"id", "execution_id", "stage", "attempt_id", "provider_session_id", "readiness_at", "replaced_session_id", "opened_at", "closed_at"}
	entryColumns      = []string{"id", "execution_id", "session_id", "attempt_id", "speaker", "ordinal", "revision", "evidence_state", "body", "cites_entry_id", "supersedes_entry_id", "committed_at"}
	lossColumns       = []string{"id", "execution_id", "question_id", "session_id", "attempt_id", "resolved_at", "created_at"}
	handoffColumns    = []string{"id", "execution_id", "source_stage", "target_stage", "revision", "summary", "created_at"}
	checkpointColumns = []string{"id", "execution_id", "session_id", "attempt_id", "through_ordinal", "created_at"}
)

// scenario holds the stored rows of one progress boundary, one slice of rows per table.
type scenario struct {
	progress, coverage, sessions, entries, losses, handoffs, checkpoints [][]driver.Value
}

// storeFixture bundles the adapter under test with its mocked collaborators.
type storeFixture struct {
	store interviewrepository.Store
	mock  sqlmock.Sqlmock
	pins  *mocks.PinReader
}

// newFixture opens the adapter over sqlmock with ping disabled so tests own the SQL expectations.
func newFixture(t *testing.T) storeFixture {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)
	pins := mocks.NewPinReader(t)
	clock := mocks.NewClock(t)
	clock.On("Now").Return(now).Maybe()
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	return storeFixture{store: interviewrepository.Store{DB: gdb, Pins: pins, Clock: clock}, mock: mock, pins: pins}
}

// hrConfiguration builds the pinned configuration without a follow-up field.
func hrConfiguration(t *testing.T) valueobject.HRConfiguration {
	t.Helper()
	c, err := valueobject.NewHRConfiguration(valueobject.HRInput{
		MandatoryQuestions: []valueobject.QuestionInput{{ID: "background", Stage: "background", Text: "Tell us about your work"}, {ID: "screening", Stage: "screening", Text: "Explain a project"}, {ID: "compensation", Stage: "compensation", Text: "What do you expect?"}},
		StageSeconds:       &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{DiscussionMode: "expectations_only"},
	})
	require.NoError(t, err)
	return c
}

// pinned returns the revision pin and lifecycle instant the mocked reader resolves.
func pinned(t *testing.T) interviewport.PinnedContext {
	t.Helper()
	return interviewport.PinnedContext{RevisionID: revisionID, Configuration: hrConfiguration(t), RunningAt: base}
}

// freshRecord creates the initial record through the domain factory with the first session shell.
func freshRecord(t *testing.T) interviewport.ProgressRecord {
	t.Helper()
	version, err := agentconfig.NewDraftVersion(1)
	require.NoError(t, err)
	revision, err := agentconfig.RestoreRevision(agentconfig.RevisionSnapshot[valueobject.HRConfiguration]{
		Identity:     agentconfig.RevisionIdentity{ID: revisionID, InstanceID: "instance", Number: 1, DraftVersion: version, Template: agentconfig.TemplateRef{Template: "hr", Version: "1", Behavior: "1"}},
		ValidationID: "publication", Configuration: hrConfiguration(t), CreatedAt: base,
	})
	require.NoError(t, err)
	p, err := interview.NewProgress(execID, revision, base)
	require.NoError(t, err)
	return interviewport.ProgressRecord{
		Snapshot: p.Snapshot(), Version: 1,
		Sessions: []interviewport.VoiceSession{{ID: sessionOne, AttemptID: attemptOne, Stage: enum.StageBackground, OpenedAt: base}},
	}
}

// pendingCoverage stores the three mandatory questions as untouched.
func pendingCoverage() [][]driver.Value {
	rows := make([][]driver.Value, 0, 3)
	for _, q := range []string{"background", "screening", "compensation"} {
		rows = append(rows, []driver.Value{"row-" + q, execID, q, q, "pending", nil, int64(0), nil, base})
	}
	return rows
}

// freshScenario is stage background at progress version 3 with every question pending.
func freshScenario() scenario {
	return scenario{
		progress: [][]driver.Value{{execID, int64(2), "background", base, int64(3), nil, base, base}},
		coverage: pendingCoverage(),
		sessions: [][]driver.Value{{sessionOne, execID, "background", attemptOne, nil, nil, nil, base, nil}},
	}
}

// readyScenario answers the background question with a final candidate entry, so a handoff may be sealed.
func readyScenario() scenario {
	sc := freshScenario()
	sc.coverage[0] = []driver.Value{"row-background", execID, "background", "background", "answered", nil, int64(0), entryOne, base}
	sc.entries = [][]driver.Value{{entryOne, execID, sessionOne, attemptOne, "candidate", int64(1), int64(1), "final", "I built payment APIs", nil, nil, base.Add(time.Second)}}
	return sc
}

// sealedScenario has a committed background handoff, a checkpoint and the screening session open.
func sealedScenario() scenario {
	sc := readyScenario()
	closed := base.Add(90 * time.Second)
	sc.progress = [][]driver.Value{{execID, int64(2), "screening", base.Add(2 * time.Minute), int64(5), nil, base, base.Add(2 * time.Minute)}}
	sc.sessions = [][]driver.Value{
		{sessionOne, execID, "background", attemptOne, "provider-1", base.Add(time.Second), nil, base, closed},
		{sessionTwo, execID, "screening", attemptTwo, nil, nil, nil, base.Add(2 * time.Minute), nil},
	}
	sc.handoffs = [][]driver.Value{{"handoff-1", execID, "background", "screening", int64(1), "Covered background", closed}}
	sc.checkpoints = [][]driver.Value{{"checkpoint-1", execID, sessionOne, attemptOne, int64(1), base.Add(time.Minute)}}
	return sc
}

// rowsOf turns column names and values into sqlmock rows.
func rowsOf(columns []string, data [][]driver.Value) *sqlmock.Rows {
	rows := sqlmock.NewRows(columns)
	for _, row := range data {
		rows.AddRow(row...)
	}
	return rows
}

// expectProgressRead expects the progress row read, locked when the caller updates.
func expectProgressRead(mock sqlmock.Sqlmock, sc scenario, lock bool) {
	pattern := `(?i)SELECT .* FROM "interview_progress" WHERE execution_id = .+ LIMIT .+`
	if lock {
		pattern += ` FOR UPDATE`
	}
	mock.ExpectQuery(pattern).WithArgs(execID, 1).WillReturnRows(rowsOf(progressColumns, sc.progress))
}

// expectChildrenRead expects the six bounded child reads that follow the progress row.
func expectChildrenRead(mock sqlmock.Sqlmock, sc scenario) {
	for _, read := range []struct {
		table   string
		columns []string
		rows    [][]driver.Value
	}{
		{"interview_coverage_items", coverageColumns, sc.coverage},
		{"interview_voice_sessions", sessionColumns, sc.sessions},
		{"interview_transcript_entries", entryColumns, sc.entries},
		{"interview_technical_losses", lossColumns, sc.losses},
		{"interview_handoffs", handoffColumns, sc.handoffs},
		{"interview_checkpoints", checkpointColumns, sc.checkpoints},
	} {
		mock.ExpectQuery(`(?i)SELECT .* FROM "` + read.table + `" WHERE execution_id = .+`).WithArgs(execID).WillReturnRows(rowsOf(read.columns, read.rows))
	}
}

// mutate applies one domain operation to the record snapshot, as an application operation would.
func mutate(t *testing.T, rec *interviewport.ProgressRecord, op func(*interview.Progress) error) {
	t.Helper()
	p, err := interview.RestoreProgress(rec.Snapshot)
	require.NoError(t, err)
	require.NoError(t, op(p))
	rec.Snapshot = p.Snapshot()
}
