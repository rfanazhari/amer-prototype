package interviewrepository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/transaction"
	"gorm.io/gorm"
)

// Insert writes initial progress at version one with its children in one transaction.
// A duplicate execution is ErrConflict; state the tables cannot hold is refused before any SQL.
func (s Store) Insert(ctx context.Context, req interviewport.ProgressWrite, record interviewport.ProgressRecord) error {
	if req.ExecutionID == "" || req.ExecutionID != record.Snapshot.ExecutionID {
		return unmappable("write request and record name different executions")
	}
	if err := representable(record); err != nil {
		return err
	}
	return transaction.Within(ctx, s.DB, func(ctx context.Context, tx *gorm.DB) error {
		at := s.instant()
		snap := record.Snapshot
		row := ProgressModel{
			ExecutionID: snap.ExecutionID, FollowUpLimit: snap.FollowUpLimit, CurrentStage: snap.CurrentStage.String(),
			StageActivatedAt: snap.StageActivatedAt.UTC(), CommittedVersion: 1, ControllerID: textPointer(record.ControllerID),
			FollowUpEntryIDs: encodeFollowUps(snap.FollowUps), ReconnectEpisodes: snap.Reconnect.Episodes, ReconnectAttempts: snap.Reconnect.Attempts,
			ReconnectDetectedAt: utcPointer(snap.Reconnect.DetectedAt), CreatedAt: at, UpdatedAt: at,
		}
		if err := tx.Create(&row).Error; err != nil {
			return classify(err)
		}
		return s.writeChildren(tx, interviewport.ProgressRecord{}, record, index{ordinals: map[string]int{}, sessions: map[string]string{}}, at)
	})
}

// Update locks progress, checks the expected version, runs change on a detached record and writes only the
// difference. Rows already committed may not be rewritten. The version moves only after every write succeeds.
func (s Store) Update(ctx context.Context, req interviewport.ProgressWrite, change func(*interviewport.ProgressRecord) error) error {
	return transaction.Within(ctx, s.DB, func(ctx context.Context, tx *gorm.DB) error {
		row, err := readProgress(tx, req.ExecutionID, true)
		if err != nil {
			return err
		}
		if row.CommittedVersion != req.ExpectedVersion {
			return interviewport.ErrConflict
		}
		before, idx, err := s.assemble(ctx, tx, row)
		if err != nil {
			return err
		}
		after := cloneRecord(before)
		if err := change(&after); err != nil {
			return err
		}
		if err := representable(after); err != nil {
			return err
		}
		if err := stable(before, after); err != nil {
			return err
		}
		at := s.instant()
		if err := s.writeChildren(tx, before, after, idx, at); err != nil {
			return err
		}
		next := ProgressModel{
			CurrentStage: after.Snapshot.CurrentStage.String(), StageActivatedAt: after.Snapshot.StageActivatedAt.UTC(),
			CommittedVersion: row.CommittedVersion + 1, ControllerID: textPointer(after.ControllerID),
			FollowUpEntryIDs: encodeFollowUps(after.Snapshot.FollowUps), ReconnectEpisodes: after.Snapshot.Reconnect.Episodes,
			ReconnectAttempts: after.Snapshot.Reconnect.Attempts, ReconnectDetectedAt: utcPointer(after.Snapshot.Reconnect.DetectedAt), UpdatedAt: at,
		}
		result := tx.Model(&ProgressModel{}).
			Where("execution_id = ? AND committed_version = ?", row.ExecutionID, row.CommittedVersion).
			Select("current_stage", "stage_activated_at", "committed_version", "controller_id", "follow_up_entry_ids",
				"reconnect_episodes", "reconnect_attempts", "reconnect_detected_at", "updated_at").
			Updates(&next)
		if result.Error != nil {
			return classify(result.Error)
		}
		if result.RowsAffected == 0 {
			return interviewport.ErrConflict
		}
		return nil
	})
}

// writeChildren persists sessions, entries, losses, coverage, checkpoints and handoffs in foreign-key order.
// Sessions change before new ones open so the single open session constraint holds inside the transaction.
func (s Store) writeChildren(tx *gorm.DB, before, after interviewport.ProgressRecord, idx index, at time.Time) error {
	id := after.Snapshot.ExecutionID
	for i, old := range before.Sessions {
		next := after.Sessions[i]
		if sameSessionState(old, next) {
			continue
		}
		model := SessionModel{ProviderSessionID: textPointer(next.ProviderSessionID), ReadinessAt: utcPointer(next.ReadinessAt), ClosedAt: utcPointer(next.ClosedAt)}
		err := tx.Model(&SessionModel{}).Where("id = ?", next.ID).Select("provider_session_id", "readiness_at", "closed_at").Updates(&model).Error
		if err != nil {
			return classify(err)
		}
	}
	if opened := after.Sessions[len(before.Sessions):]; len(opened) > 0 {
		models := make([]SessionModel, 0, len(opened))
		for _, v := range opened {
			models = append(models, SessionModel{
				ID: v.ID, ExecutionID: id, Stage: v.Stage.String(), AttemptID: v.AttemptID, ProviderSessionID: textPointer(v.ProviderSessionID),
				ReadinessAt: utcPointer(v.ReadinessAt), ReplacedSessionID: textPointer(v.ReplacedSessionID), OpenedAt: v.OpenedAt.UTC(), ClosedAt: utcPointer(v.ClosedAt),
			})
		}
		if err := tx.Create(&models).Error; err != nil {
			return classify(err)
		}
	}
	if err := s.writeEntries(tx, before, after, idx, at); err != nil {
		return err
	}
	if err := s.writeLosses(tx, before, after, at); err != nil {
		return err
	}
	if err := writeCoverage(tx, before, after, at); err != nil {
		return err
	}
	return writeSealed(tx, before, after, idx)
}

// sameSessionState compares the three session columns that may change after a session opens.
func sameSessionState(a, b interviewport.VoiceSession) bool {
	return equalText(a.ProviderSessionID, b.ProviderSessionID) && equalTime(a.ReadinessAt, b.ReadinessAt) && equalTime(a.ClosedAt, b.ClosedAt)
}

// writeEntries appends newly classified entries to the latest session of their stage, in ordinal order.
func (s Store) writeEntries(tx *gorm.DB, before, after interviewport.ProgressRecord, idx index, at time.Time) error {
	var models []EntryModel
	for _, entry := range after.Snapshot.Entries[len(before.Snapshot.Entries):] {
		session, ok := stageSession(after.Sessions, entry.Stage)
		if !ok {
			return unmappable("entry %s has no session for its stage", entry.ID)
		}
		ordinal := 1
		for _, seen := range idx.sessions {
			if seen == session.ID {
				ordinal++
			}
		}
		idx.ordinals[entry.ID], idx.sessions[entry.ID] = ordinal, session.ID
		models = append(models, EntryModel{
			ID: entry.ID, ExecutionID: after.Snapshot.ExecutionID, SessionID: session.ID, AttemptID: session.AttemptID,
			Speaker: string(entry.Speaker), Ordinal: ordinal, Revision: 1, EvidenceState: string(entry.State), Body: entry.Body, CommittedAt: at,
		})
	}
	if len(models) == 0 {
		return nil
	}
	return classify(tx.Create(&models).Error)
}

// writeLosses appends new technical losses and stamps the resolution of existing ones.
func (s Store) writeLosses(tx *gorm.DB, before, after interviewport.ProgressRecord, at time.Time) error {
	stages := make(map[string]string, len(after.Snapshot.Coverage))
	for _, q := range after.Snapshot.Coverage {
		stages[q.QuestionID] = q.Stage.String()
	}
	for i, old := range before.Snapshot.Losses {
		if next := after.Snapshot.Losses[i]; next.Resolved && !old.Resolved {
			err := tx.Model(&LossModel{}).Where("id = ?", next.ID).Select("resolved_at").Updates(&LossModel{ResolvedAt: &at}).Error
			if err != nil {
				return classify(err)
			}
		}
	}
	for _, loss := range after.Snapshot.Losses[len(before.Snapshot.Losses):] {
		stage, known := stages[loss.QuestionID]
		session, ok := stageSession(after.Sessions, enum.QuestionStage(stage))
		if !known || !ok {
			return unmappable("loss %s has no question or session for its stage", loss.ID)
		}
		model := LossModel{ID: loss.ID, ExecutionID: after.Snapshot.ExecutionID, QuestionID: loss.QuestionID, SessionID: session.ID, AttemptID: session.AttemptID, CreatedAt: at}
		if loss.Resolved {
			model.ResolvedAt = &at
		}
		if err := tx.Create(&model).Error; err != nil {
			return classify(err)
		}
	}
	return nil
}

// writeCoverage creates the pending inventory once and afterwards updates only changed items by explicit columns.
func writeCoverage(tx *gorm.DB, before, after interviewport.ProgressRecord, at time.Time) error {
	if len(before.Snapshot.Coverage) == 0 {
		models := make([]CoverageModel, 0, len(after.Snapshot.Coverage))
		for _, q := range after.Snapshot.Coverage {
			models = append(models, coverageModel(after.Snapshot.ExecutionID, uuid.NewString(), q, at))
		}
		if len(models) == 0 {
			return nil
		}
		return classify(tx.Create(&models).Error)
	}
	for i, old := range before.Snapshot.Coverage {
		q := after.Snapshot.Coverage[i]
		if q == old {
			continue
		}
		model := coverageModel(after.Snapshot.ExecutionID, "", q, at)
		result := tx.Model(&CoverageModel{}).
			Where("execution_id = ? AND question_id = ?", model.ExecutionID, q.QuestionID).
			Select("state", "omission_reason", "clarification_count", "answer_entry_id", "clarification_entry_id", "omission_entry_id", "updated_at").
			Updates(&model)
		if result.Error != nil {
			return classify(result.Error)
		}
		if result.RowsAffected == 0 {
			return unmappable("coverage row %s is missing", q.QuestionID)
		}
	}
	return nil
}

// writeSealed appends checkpoints and handoffs after the rows they reference exist.
func writeSealed(tx *gorm.DB, before, after interviewport.ProgressRecord, idx index) error {
	for _, cp := range after.Checkpoints[len(before.Checkpoints):] {
		session, ok := sessionByID(after.Sessions, cp.SessionID)
		if !ok || session.Stage != cp.Stage || session.AttemptID != cp.AttemptID {
			return unmappable("checkpoint %s does not match its session", cp.ID)
		}
		through := 0
		for _, entryID := range cp.EntryIDs {
			if idx.sessions[entryID] != cp.SessionID {
				return unmappable("checkpoint %s cites an entry of another session", cp.ID)
			}
			through = max(through, idx.ordinals[entryID])
		}
		model := CheckpointModel{ID: cp.ID, ExecutionID: after.Snapshot.ExecutionID, SessionID: cp.SessionID, AttemptID: cp.AttemptID, ThroughOrdinal: through, CreatedAt: cp.CommittedAt.UTC()}
		if err := tx.Create(&model).Error; err != nil {
			return classify(err)
		}
	}
	for i := len(before.Snapshot.Handoffs); i < len(after.Snapshot.Handoffs); i++ {
		h := after.Snapshot.Handoffs[i]
		model := HandoffModel{ID: uuid.NewString(), ExecutionID: after.Snapshot.ExecutionID, SourceStage: h.Stage.String(), Revision: i + 1, CreatedAt: h.CommittedAt.UTC()}
		if h.Target != "" {
			target := h.Target.String()
			model.TargetStage = &target
		}
		if summary, ok := after.HandoffSummaries[h.Stage]; ok {
			model.Summary = &summary
		}
		if err := tx.Create(&model).Error; err != nil {
			return classify(err)
		}
	}
	return nil
}
