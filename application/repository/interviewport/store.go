// Package interviewport defines storage boundaries for interview-owned progress.
package interviewport

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

var (
	// ErrNotFound reports absent interview progress.
	ErrNotFound = errors.New("interview progress not found")
	// ErrConflict reports a stale progress version or an existing execution progress record.
	ErrConflict = errors.New("interview progress version conflict")
)

// VoiceSession transfers local session state; mutable time pointers must be detached by adapters.
type VoiceSession struct {
	ID                string
	AttemptID         string
	Stage             enum.QuestionStage
	ProviderSessionID *string
	ReadinessAt       *time.Time
	ReplacedSessionID *string
	OpenedAt          time.Time
	ClosedAt          *time.Time
}

// Checkpoint identifies committed in-stage evidence usable for recovery.
type Checkpoint struct {
	ID          string
	Stage       enum.QuestionStage
	SessionID   string
	AttemptID   string
	EntryIDs    []string
	CommittedAt time.Time
}

// ProgressRecord is a detached domain snapshot with persistence coordination metadata.
// Snapshot owns coverage, classified transcript, technical losses and structured handoffs.
// Optional summaries are keyed by source stage; absence means no narrative, not no handoff.
type ProgressRecord struct {
	Snapshot         interview.Snapshot
	Version          int64
	ControllerID     *string
	Sessions         []VoiceSession
	Checkpoints      []Checkpoint
	HandoffSummaries map[enum.QuestionStage]string
}

// ProgressWrite correlates a storage mutation with lifecycle authority and progress version.
// ExpectedVersion is the progress version, not the lifecycle execution version.
// The store never replaces the lifecycle's current-attempt check.
type ProgressWrite struct {
	ExecutionID     string
	AttemptID       string
	ExpectedVersion int64
}

// ProgressStore persists interview progress inside the caller's transaction context.
// No SQL or GORM handle crosses the port. Every read and write returns detached values.
// Corrupt restored state is a technical error wrapping interview.ErrRestore.
type ProgressStore interface {
	// Insert writes initial progress (version 1) and child records atomically; duplicate execution
	// yields ErrConflict. The caller first obtains lifecycle authority using WithProgress.
	Insert(ctx context.Context, req ProgressWrite, record ProgressRecord) error
	// Find reads one complete progress boundary without mutation; absence yields ErrNotFound.
	Find(ctx context.Context, executionID string) (ProgressRecord, error)
	// Update locks progress and checks ExpectedVersion before invoking change. It writes all changed
	// children atomically and increments Version only after success. Callback failure rolls back.
	// It joins the lifecycle transaction carried by ctx; execution must be locked before child rows.
	// A handoff and source-session close commit together; target create happens after commit.
	Update(ctx context.Context, req ProgressWrite, change func(*ProgressRecord) error) error
}
