package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewrepository"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// progressFixture holds one execution pin plus the interview store over the release schema.
type progressFixture struct {
	store       interviewrepository.Store
	db          *gorm.DB
	executionID string
	revisionID  string
	running     time.Time
	sessionID   string
	attemptID   string
}

// interviewTables lists the children deleted before the progress row, in foreign-key order.
var interviewTables = []string{
	"interview_checkpoints", "interview_technical_losses", "interview_session_commands",
	"interview_transcript_entries", "interview_handoffs", "interview_voice_sessions", "interview_coverage_items",
}

// newProgressFixture admits an execution pin and wires the store to a mocked pin reader and clock.
func newProgressFixture(t *testing.T) progressFixture {
	t.Helper()
	base := newExecutionFixture(t)
	snap, fact := base.admit(t)
	require.NoError(t, base.store.InsertInterview(context.Background(), base.assignmentID, snap, fact))
	running := time.Now().UTC().Truncate(time.Millisecond)
	config, err := valueobject.NewHRConfiguration(valueobject.HRInput{
		MandatoryQuestions: []valueobject.QuestionInput{{ID: "background", Stage: "background", Text: "Tell us about your work"}, {ID: "screening", Stage: "screening", Text: "Explain a project"}, {ID: "compensation", Stage: "compensation", Text: "What do you expect?"}},
		StageSeconds:       &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{DiscussionMode: "expectations_only"},
	})
	require.NoError(t, err)
	pins := mocks.NewPinReader(t)
	pins.On("Find", mock.Anything, snap.ID).Return(interviewport.PinnedContext{RevisionID: snap.Binding.RevisionID, Configuration: config, RunningAt: running}, nil).Maybe()
	clock := mocks.NewClock(t)
	clock.On("Now").Return(running.Add(time.Minute)).Maybe()
	t.Cleanup(func() {
		for _, table := range interviewTables {
			require.NoError(t, base.db.Exec(`DELETE FROM `+table+` WHERE execution_id = ?`, snap.ID).Error)
		}
		require.NoError(t, base.db.Exec(`DELETE FROM interview_progress WHERE execution_id = ?`, snap.ID).Error)
	})
	return progressFixture{
		store: interviewrepository.Store{DB: base.db, Pins: pins, Clock: clock}, db: base.db, executionID: snap.ID,
		revisionID: snap.Binding.RevisionID, running: running, sessionID: uuid.NewString(), attemptID: uuid.NewString(),
	}
}

// record builds the initial domain record and first session shell through the domain factory.
func (f progressFixture) record(t *testing.T) interviewport.ProgressRecord {
	t.Helper()
	pin, err := f.store.Pins.Find(context.Background(), f.executionID)
	require.NoError(t, err)
	version, err := agentconfig.NewDraftVersion(1)
	require.NoError(t, err)
	revision, err := agentconfig.RestoreRevision(agentconfig.RevisionSnapshot[valueobject.HRConfiguration]{
		Identity:     agentconfig.RevisionIdentity{ID: f.revisionID, InstanceID: "instance", Number: 1, DraftVersion: version, Template: agentconfig.TemplateRef{Template: "hr", Version: "1", Behavior: "1"}},
		ValidationID: "publication", Configuration: pin.Configuration, CreatedAt: f.running,
	})
	require.NoError(t, err)
	p, err := interview.NewProgress(f.executionID, revision, f.running)
	require.NoError(t, err)
	return interviewport.ProgressRecord{
		Snapshot: p.Snapshot(), Version: 1,
		Sessions: []interviewport.VoiceSession{{ID: f.sessionID, AttemptID: f.attemptID, Stage: enum.StageBackground, OpenedAt: f.running}},
	}
}

// write names the progress write for one expected version.
func (f progressFixture) write(version int64) interviewport.ProgressWrite {
	return interviewport.ProgressWrite{ExecutionID: f.executionID, AttemptID: f.attemptID, ExpectedVersion: version}
}

// answerBackground asks, records a final candidate entry and answers the background question.
func (f progressFixture) answerBackground(t *testing.T, rec *interviewport.ProgressRecord) {
	t.Helper()
	p, err := interview.RestoreProgress(rec.Snapshot)
	require.NoError(t, err)
	entry := uuid.NewString()
	require.NoError(t, p.Ask("background"))
	require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: entry, Stage: enum.StageBackground, Speaker: interview.Candidate, State: interview.Final, Body: "I built payment APIs"}))
	require.NoError(t, p.Answer("background", entry))
	rec.Snapshot = p.Snapshot()
}

// seal commits the background handoff and closes the source session at one instant.
func (f progressFixture) seal(t *testing.T, rec *interviewport.ProgressRecord) {
	t.Helper()
	at := f.running.Add(2 * time.Minute)
	p, err := interview.RestoreProgress(rec.Snapshot)
	require.NoError(t, err)
	require.NoError(t, p.CommitHandoff(enum.StageScreening, at))
	rec.Snapshot = p.Snapshot()
	rec.Sessions[0].ClosedAt = &at
}

// count returns the number of rows a query reports.
func (f progressFixture) count(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Raw(query, args...).Scan(&n).Error)
	return n
}

// pgCode extracts the PostgreSQL error code from a driver failure.
func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// progressStoreCase names one real PostgreSQL progress store proof.
type progressStoreCase struct {
	name   string
	verify func(t *testing.T, f progressFixture)
}

// TestInterviewProgressStorePostgres proves round trips, explicit zero and null writes, uniqueness, rollback and corrupt restore.
func TestInterviewProgressStorePostgres(t *testing.T) {
	tests := []progressStoreCase{
		{name: "insert, answer, seal and reload round trip with explicit zero and null columns", verify: proveProgressRoundTrip},
		{name: "second open session is refused by PostgreSQL and by the store", verify: proveOneOpenSession},
		{name: "second handoff for one source stage is refused by PostgreSQL", verify: proveOneHandoffPerStage},
		{name: "failed write after the source session close rolls the close back", verify: proveHandoffRollback},
		{name: "corrupt stored coverage is a technical restore error", verify: proveCorruptRestore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.verify(t, newProgressFixture(t)) })
	}
}

// proveProgressRoundTrip persists progress, an answer and a sealed handoff, then restores them.
func proveProgressRoundTrip(t *testing.T, f progressFixture) {
	ctx := context.Background()
	rec := f.record(t)
	require.NoError(t, f.store.Insert(ctx, f.write(0), rec))
	require.EqualValues(t, 3, f.count(t, `SELECT count(*) FROM interview_coverage_items WHERE execution_id = ? AND clarification_count = 0 AND answer_entry_id IS NULL AND omission_reason IS NULL`, f.executionID))

	require.ErrorIs(t, f.store.Insert(ctx, f.write(0), rec), interviewport.ErrConflict)

	loaded, err := f.store.Find(ctx, f.executionID)
	require.NoError(t, err)
	require.EqualValues(t, 1, loaded.Version)
	f.answerBackground(t, &loaded)
	require.NoError(t, f.store.Update(ctx, f.write(1), func(r *interviewport.ProgressRecord) error { *r = loaded; return nil }))

	loaded, err = f.store.Find(ctx, f.executionID)
	require.NoError(t, err)
	require.EqualValues(t, 2, loaded.Version)
	f.seal(t, &loaded)
	require.NoError(t, f.store.Update(ctx, f.write(2), func(r *interviewport.ProgressRecord) error { *r = loaded; return nil }))

	loaded, err = f.store.Find(ctx, f.executionID)
	require.NoError(t, err)
	require.EqualValues(t, 3, loaded.Version)
	require.Equal(t, enum.StageBackground, loaded.Snapshot.CurrentStage)
	require.Len(t, loaded.Snapshot.Handoffs, 1)
	require.NotNil(t, loaded.Sessions[0].ClosedAt)
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ? AND summary IS NULL AND target_stage = 'screening'`, f.executionID))
	require.EqualValues(t, 0, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND closed_at IS NULL`, f.executionID))
}

// proveOneOpenSession shows a second open session fails at the database and through the store.
func proveOneOpenSession(t *testing.T, f progressFixture) {
	ctx := context.Background()
	require.NoError(t, f.store.Insert(ctx, f.write(0), f.record(t)))

	err := f.db.Exec(`INSERT INTO interview_voice_sessions (id, execution_id, stage, attempt_id, opened_at) VALUES (?, ?, 'background', ?, now())`,
		uuid.NewString(), f.executionID, uuid.NewString()).Error
	require.Equal(t, "23505", pgCode(err))

	err = f.store.Update(ctx, f.write(1), func(r *interviewport.ProgressRecord) error {
		r.Sessions = append(r.Sessions, interviewport.VoiceSession{ID: uuid.NewString(), AttemptID: uuid.NewString(), Stage: enum.StageBackground, OpenedAt: f.running.Add(time.Second)})
		return nil
	})
	require.ErrorIs(t, err, interviewport.ErrConflict)
}

// proveOneHandoffPerStage shows PostgreSQL refuses a duplicate source-stage handoff.
func proveOneHandoffPerStage(t *testing.T, f progressFixture) {
	ctx := context.Background()
	rec := f.record(t)
	require.NoError(t, f.store.Insert(ctx, f.write(0), rec))
	loaded, err := f.store.Find(ctx, f.executionID)
	require.NoError(t, err)
	f.answerBackground(t, &loaded)
	f.seal(t, &loaded)
	require.NoError(t, f.store.Update(ctx, f.write(1), func(r *interviewport.ProgressRecord) error { *r = loaded; return nil }))

	err = f.db.Exec(`INSERT INTO interview_handoffs (id, execution_id, source_stage, target_stage, revision, created_at) VALUES (?, ?, 'background', 'screening', 2, now())`,
		uuid.NewString(), f.executionID).Error
	require.Equal(t, "23505", pgCode(err))
}

// proveHandoffRollback forces a duplicate provider session id after the source session close and shows nothing persisted.
func proveHandoffRollback(t *testing.T, f progressFixture) {
	ctx := context.Background()
	require.NoError(t, f.store.Insert(ctx, f.write(0), f.record(t)))
	loaded, err := f.store.Find(ctx, f.executionID)
	require.NoError(t, err)
	f.answerBackground(t, &loaded)
	provider := "provider-" + uuid.NewString()
	loaded.Sessions[0].ProviderSessionID = &provider
	require.NoError(t, f.store.Update(ctx, f.write(1), func(r *interviewport.ProgressRecord) error { *r = loaded; return nil }))

	loaded, err = f.store.Find(ctx, f.executionID)
	require.NoError(t, err)
	f.seal(t, &loaded)
	loaded.Sessions = append(loaded.Sessions, interviewport.VoiceSession{
		ID: uuid.NewString(), AttemptID: uuid.NewString(), Stage: enum.StageScreening, ProviderSessionID: &provider, OpenedAt: *loaded.Sessions[0].ClosedAt,
	})
	err = f.store.Update(ctx, f.write(2), func(r *interviewport.ProgressRecord) error { *r = loaded; return nil })

	require.ErrorIs(t, err, interviewport.ErrConflict)
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND closed_at IS NULL`, f.executionID))
	require.EqualValues(t, 0, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ?`, f.executionID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_progress WHERE execution_id = ? AND current_stage = 'background' AND committed_version = 2`, f.executionID))
}

// proveCorruptRestore shows a row that passes the table checks but breaks the answer citation rule is not served.
func proveCorruptRestore(t *testing.T, f progressFixture) {
	ctx := context.Background()
	require.NoError(t, f.store.Insert(ctx, f.write(0), f.record(t)))
	require.NoError(t, f.db.Exec(`UPDATE interview_coverage_items SET state = 'answered' WHERE execution_id = ? AND question_id = 'background'`, f.executionID).Error)

	_, err := f.store.Find(ctx, f.executionID)

	require.ErrorIs(t, err, interview.ErrRestore)
	require.NotErrorIs(t, err, interviewport.ErrNotFound)
}
