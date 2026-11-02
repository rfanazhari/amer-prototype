package interviewrepository

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// unmappable wraps ErrUnmappable with the reason the schema cannot hold the state.
func unmappable(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrUnmappable, fmt.Sprintf(format, args...))
}

// representable refuses sessions the tables cannot hold, so state is never silently dropped.
func representable(rec interviewport.ProgressRecord) error {
	for _, session := range rec.Sessions {
		if session.ID == "" || session.AttemptID == "" || !session.Stage.Valid() {
			return unmappable("session needs an id, an attempt id and a known stage")
		}
	}
	return nil
}

// stable verifies that rows already committed were not rewritten; only appends and the allowed state moves persist.
func stable(before, after interviewport.ProgressRecord) error {
	b, a := before.Snapshot, after.Snapshot
	if b.ExecutionID != a.ExecutionID || b.RevisionID != a.RevisionID || b.FollowUpLimit != a.FollowUpLimit || !b.RunningAt.Equal(a.RunningAt) {
		return unmappable("identity, pin or follow-up cap changed")
	}
	if len(a.Coverage) != len(b.Coverage) || len(a.Entries) < len(b.Entries) || len(a.Losses) < len(b.Losses) ||
		len(a.Handoffs) < len(b.Handoffs) || len(after.Sessions) < len(before.Sessions) || len(after.Checkpoints) < len(before.Checkpoints) {
		return unmappable("committed rows were removed")
	}
	for i, q := range b.Coverage {
		if a.Coverage[i].QuestionID != q.QuestionID || a.Coverage[i].Stage != q.Stage {
			return unmappable("coverage inventory changed")
		}
	}
	for i, loss := range b.Losses {
		next := a.Losses[i]
		if next.ID != loss.ID || next.QuestionID != loss.QuestionID || (loss.Resolved && !next.Resolved) {
			return unmappable("technical loss rewritten")
		}
	}
	for i, h := range b.Handoffs {
		if a.Handoffs[i].Stage != h.Stage || a.Handoffs[i].Target != h.Target || !a.Handoffs[i].CommittedAt.Equal(h.CommittedAt) ||
			after.HandoffSummaries[h.Stage] != before.HandoffSummaries[h.Stage] {
			return unmappable("committed handoff rewritten")
		}
	}
	if !slices.Equal(b.FollowUps, a.FollowUps[:min(len(b.FollowUps), len(a.FollowUps))]) || len(a.FollowUps) < len(b.FollowUps) {
		return unmappable("committed follow-up citations rewritten")
	}
	if !slices.Equal(b.Entries, a.Entries[:len(b.Entries)]) {
		return unmappable("committed transcript rewritten")
	}
	for i, cp := range before.Checkpoints {
		if !reflect.DeepEqual(cp, after.Checkpoints[i]) {
			return unmappable("committed checkpoint rewritten")
		}
	}
	for i, session := range before.Sessions {
		next := after.Sessions[i]
		if next.ID != session.ID || next.AttemptID != session.AttemptID || next.Stage != session.Stage || !next.OpenedAt.Equal(session.OpenedAt) ||
			!reflect.DeepEqual(next.ReplacedSessionID, session.ReplacedSessionID) {
			return unmappable("session identity rewritten")
		}
	}
	return nil
}

// stageSession finds the latest session opened for a stage, the session that owns its evidence.
func stageSession(sessions []interviewport.VoiceSession, stage enum.QuestionStage) (interviewport.VoiceSession, bool) {
	for i := len(sessions) - 1; i >= 0; i-- {
		if sessions[i].Stage == stage {
			return sessions[i], true
		}
	}
	return interviewport.VoiceSession{}, false
}
