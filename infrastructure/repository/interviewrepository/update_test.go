package interviewrepository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// errProbe forces a callback failure after the record was changed.
var errProbe = errors.New("rollback probe")

// updateCase names one locked read-modify-write of a progress boundary.
type updateCase struct {
	name     string
	scenario scenario
	expected int64
	change   func(t *testing.T) func(*interviewport.ProgressRecord) error
	expect   func(f storeFixture)
	wantErr  error
}

// expectLockedLoad expects the transaction, the locking progress read, the pin lookup and the child reads.
func expectLockedLoad(f storeFixture, t *testing.T, sc scenario) {
	f.mock.ExpectBegin()
	expectProgressRead(f.mock, sc, true)
	f.pins.On("Find", mock.Anything, execID).Return(pinned(t), nil).Once()
	expectChildrenRead(f.mock, sc)
}

// expectProgressBump expects the version-guarded progress update with explicit nullable columns.
func expectProgressBump(f storeFixture, stage string, from int64, affected int64) {
	expectProgressWith(f, stage, from, affected, "[]", 0, 0, nil)
}

// expectProgressWith expects the guarded progress update with explicit follow-up and reconnect columns.
func expectProgressWith(f storeFixture, stage string, from, affected int64, followUps string, episodes, attempts int, detected any) {
	f.mock.ExpectExec(`(?i)UPDATE "interview_progress" SET .+ WHERE execution_id = .+ AND committed_version = .+`).
		WithArgs(stage, base, from+1, nil, followUps, episodes, attempts, detected, now, execID, from).
		WillReturnResult(sqlmock.NewResult(0, affected))
}

// TestStoreUpdate proves version guarding, rollback, explicit zero and null writes, and sealed handoffs.
func TestStoreUpdate(t *testing.T) {
	closedAt := base.Add(5 * time.Minute)
	tests := []updateCase{
		{
			name: "absent progress is not found", scenario: scenario{}, expected: 3, wantErr: interviewport.ErrNotFound,
			change: func(*testing.T) func(*interviewport.ProgressRecord) error {
				return func(*interviewport.ProgressRecord) error { return errProbe }
			},
			expect: func(f storeFixture) {
				f.mock.ExpectBegin()
				expectProgressRead(f.mock, scenario{}, true)
				f.mock.ExpectRollback()
			},
		},
		{
			name: "stale expected version is a conflict and the callback never runs", scenario: freshScenario(), expected: 2, wantErr: interviewport.ErrConflict,
			change: func(*testing.T) func(*interviewport.ProgressRecord) error {
				return func(*interviewport.ProgressRecord) error { return errProbe }
			},
			expect: func(f storeFixture) {
				f.mock.ExpectBegin()
				expectProgressRead(f.mock, freshScenario(), true)
				f.mock.ExpectRollback()
			},
		},
		{
			name: "callback failure after sealing a handoff rolls back and writes nothing", scenario: readyScenario(), expected: 3, wantErr: errProbe,
			change: func(t *testing.T) func(*interviewport.ProgressRecord) error {
				return func(rec *interviewport.ProgressRecord) error {
					mutate(t, rec, func(p *interview.Progress) error { return p.CommitHandoff(enum.StageScreening, closedAt) })
					return errProbe
				}
			},
			expect: func(f storeFixture) {
				expectLockedLoad(f, t, readyScenario())
				f.mock.ExpectRollback()
			},
		},
		{
			name: "asked question persists explicit zero clarification count and null columns", scenario: freshScenario(), expected: 3,
			change: func(t *testing.T) func(*interviewport.ProgressRecord) error {
				return func(rec *interviewport.ProgressRecord) error {
					mutate(t, rec, func(p *interview.Progress) error { return p.Ask("background") })
					return nil
				}
			},
			expect: func(f storeFixture) {
				expectLockedLoad(f, t, freshScenario())
				f.mock.ExpectExec(`(?i)UPDATE "interview_coverage_items" SET .+ WHERE execution_id = .+ AND question_id = .+`).
					WithArgs("asked", nil, 0, nil, nil, nil, now, execID, "background").
					WillReturnResult(sqlmock.NewResult(0, 1))
				expectProgressBump(f, "background", 3, 1)
				f.mock.ExpectCommit()
			},
		},
		{
			name: "lost version race on the progress row is a conflict", scenario: freshScenario(), expected: 3, wantErr: interviewport.ErrConflict,
			change: func(t *testing.T) func(*interviewport.ProgressRecord) error {
				return func(rec *interviewport.ProgressRecord) error {
					mutate(t, rec, func(p *interview.Progress) error { return p.Ask("background") })
					return nil
				}
			},
			expect: func(f storeFixture) {
				expectLockedLoad(f, t, freshScenario())
				f.mock.ExpectExec(`(?i)UPDATE "interview_coverage_items"`).WillReturnResult(sqlmock.NewResult(0, 1))
				expectProgressBump(f, "background", 3, 0)
				f.mock.ExpectRollback()
			},
		},
		{
			name: "handoff closes the source session and leaves the source stage current", scenario: readyScenario(), expected: 3,
			change: func(t *testing.T) func(*interviewport.ProgressRecord) error {
				return func(rec *interviewport.ProgressRecord) error {
					mutate(t, rec, func(p *interview.Progress) error { return p.CommitHandoff(enum.StageScreening, closedAt) })
					rec.Sessions[0].ClosedAt = &closedAt
					return nil
				}
			},
			expect: func(f storeFixture) {
				expectLockedLoad(f, t, readyScenario())
				f.mock.ExpectExec(`(?i)UPDATE "interview_voice_sessions" SET .+ WHERE id = .+`).
					WithArgs(nil, nil, closedAt, sessionOne).
					WillReturnResult(sqlmock.NewResult(0, 1))
				f.mock.ExpectExec(`(?i)INSERT INTO "interview_handoffs"`).
					WithArgs(sqlmock.AnyArg(), execID, "background", "screening", 1, nil, closedAt).
					WillReturnResult(sqlmock.NewResult(0, 1))
				expectProgressBump(f, "background", 3, 1)
				f.mock.ExpectCommit()
			},
		},
		{
			name: "clarification persists count one and the model entry id", scenario: readyScenario(), expected: 3,
			change: func(t *testing.T) func(*interviewport.ProgressRecord) error {
				return func(rec *interviewport.ProgressRecord) error {
					mutate(t, rec, func(p *interview.Progress) error { return p.Clarify("background", entryTwo) })
					return nil
				}
			},
			expect: func(f storeFixture) {
				expectLockedLoad(f, t, askedModelScenario())
				f.mock.ExpectExec(`(?i)UPDATE "interview_coverage_items" SET .+ WHERE execution_id = .+ AND question_id = .+`).
					WithArgs("asked", nil, 1, nil, entryTwo, nil, now, execID, "background").
					WillReturnResult(sqlmock.NewResult(0, 1))
				expectProgressBump(f, "background", 3, 1)
				f.mock.ExpectCommit()
			},
		},
		{
			name: "follow-up persists the cited candidate entry id", scenario: readyScenario(), expected: 3,
			change: func(t *testing.T) func(*interviewport.ProgressRecord) error {
				return func(rec *interviewport.ProgressRecord) error {
					mutate(t, rec, func(p *interview.Progress) error { return p.FollowUp(entryOne) })
					return nil
				}
			},
			expect: func(f storeFixture) {
				expectLockedLoad(f, t, readyScenario())
				expectProgressWith(f, "background", 3, 1, `["`+entryOne+`"]`, 0, 0, nil)
				f.mock.ExpectCommit()
			},
		},
		{
			name: "reconnect episode persists counters and the detection instant", scenario: freshScenario(), expected: 3,
			change: func(t *testing.T) func(*interviewport.ProgressRecord) error {
				return func(rec *interviewport.ProgressRecord) error {
					mutate(t, rec, func(p *interview.Progress) error { return p.BeginReconnect(base.Add(time.Minute)) })
					return nil
				}
			},
			expect: func(f storeFixture) {
				expectLockedLoad(f, t, freshScenario())
				expectProgressWith(f, "background", 3, 1, "[]", 1, 0, base.Add(time.Minute))
				f.mock.ExpectCommit()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.expect(f)

			err := f.store.Update(context.Background(), interviewport.ProgressWrite{ExecutionID: execID, AttemptID: attemptOne, ExpectedVersion: tt.expected}, tt.change(t))

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
