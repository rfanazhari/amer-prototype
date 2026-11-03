package interviewrepository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/transaction"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// index keeps the storage-only facts of a loaded boundary that the port record does not carry.
type index struct {
	ordinals map[string]int
	sessions map[string]string
}

// restoreError marks corrupt stored state as a technical restoration failure.
func restoreError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", interview.ErrRestore, fmt.Sprintf(format, args...))
}

// utc normalizes a stored instant to the repository's UTC representation.
func utc(at time.Time) time.Time { return at.UTC() }

// utcPointer detaches a nullable stored instant and normalizes it to UTC.
func utcPointer(at *time.Time) *time.Time {
	if at == nil {
		return nil
	}
	copied := utc(*at)
	return &copied
}

// textPointer detaches a nullable stored string.
func textPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

// text reads a nullable stored string as an empty string when absent.
func text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// decodeFollowUps restores the stored follow-up citations; malformed JSON is corrupt state.
func decodeFollowUps(stored string) ([]string, error) {
	var ids []string
	if err := json.Unmarshal([]byte(stored), &ids); err != nil {
		return nil, restoreError("follow-up citations are not a JSON array of ids")
	}
	return ids, nil
}

// encodeFollowUps stores follow-up citations as a JSON array, empty when none exist.
func encodeFollowUps(ids []string) string {
	if len(ids) == 0 {
		return "[]"
	}
	encoded, _ := json.Marshal(ids)
	return string(encoded)
}

// Find reads a complete progress boundary in a standalone repeatable-read transaction or joins the caller's locked unit of work.
func (s Store) Find(ctx context.Context, executionID string) (interviewport.ProgressRecord, error) {
	var record interviewport.ProgressRecord
	err := transaction.Within(ctx, s.DB, func(ctx context.Context, tx *gorm.DB) error {
		row, err := readProgress(tx, executionID, false)
		if err != nil {
			return err
		}
		record, _, err = s.assemble(ctx, tx, row)
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return interviewport.ProgressRecord{}, err
	}
	return record, nil
}

// readProgress reads the progress row, locking it for update when the caller will write.
func readProgress(tx *gorm.DB, executionID string, lock bool) (ProgressModel, error) {
	var row ProgressModel
	query := tx
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.Where("execution_id = ?", executionID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ProgressModel{}, interviewport.ErrNotFound
	}
	return row, err
}

// assemble loads the bounded child reads, resolves the pin and restores the aggregate to validate it.
func (s Store) assemble(ctx context.Context, tx *gorm.DB, row ProgressModel) (interviewport.ProgressRecord, index, error) {
	pin, err := s.Pins.Find(ctx, row.ExecutionID)
	if err != nil {
		return interviewport.ProgressRecord{}, index{}, err
	}
	var (
		coverage    []CoverageModel
		sessions    []SessionModel
		entries     []EntryModel
		losses      []LossModel
		handoffs    []HandoffModel
		checkpoints []CheckpointModel
	)
	reads := []struct {
		dest  any
		order string
	}{
		{&coverage, ""}, {&sessions, "opened_at, id"}, {&entries, "session_id, ordinal, revision"},
		{&losses, "created_at, id"}, {&handoffs, "revision"}, {&checkpoints, "created_at, id"},
	}
	for _, read := range reads {
		query := tx.Where("execution_id = ?", row.ExecutionID)
		if read.order != "" {
			query = query.Order(read.order)
		}
		if err := query.Find(read.dest).Error; err != nil {
			return interviewport.ProgressRecord{}, index{}, err
		}
	}

	record := interviewport.ProgressRecord{Version: row.CommittedVersion, ControllerID: textPointer(row.ControllerID)}
	idx := index{ordinals: make(map[string]int, len(entries)), sessions: make(map[string]string, len(entries))}
	followUps, err := decodeFollowUps(row.FollowUpEntryIDs)
	if err != nil {
		return interviewport.ProgressRecord{}, index{}, err
	}
	snap := interview.Snapshot{
		ExecutionID: row.ExecutionID, RevisionID: pin.RevisionID, Configuration: pin.Configuration.Clone(),
		RunningAt: utc(pin.RunningAt), StageActivatedAt: utc(row.StageActivatedAt),
		CurrentStage: enum.QuestionStage(row.CurrentStage), FollowUpLimit: row.FollowUpLimit, FollowUps: followUps,
		Reconnect: interview.ReconnectState{DetectedAt: utcPointer(row.ReconnectDetectedAt), Episodes: row.ReconnectEpisodes, Attempts: row.ReconnectAttempts},
	}
	stageOf, order, err := mapSessions(sessions, &record)
	if err != nil {
		return interviewport.ProgressRecord{}, index{}, err
	}
	if snap.Coverage, err = mapCoverage(coverage, pin.Configuration.MandatoryQuestions()); err != nil {
		return interviewport.ProgressRecord{}, index{}, err
	}
	if snap.Entries, err = mapEntries(entries, stageOf, order, &idx); err != nil {
		return interviewport.ProgressRecord{}, index{}, err
	}
	for _, loss := range losses {
		snap.Losses = append(snap.Losses, interview.TechnicalLoss{ID: loss.ID, QuestionID: loss.QuestionID, Resolved: loss.ResolvedAt != nil})
	}
	if err := mapHandoffs(handoffs, &snap, &record); err != nil {
		return interviewport.ProgressRecord{}, index{}, err
	}
	if err := mapCheckpoints(checkpoints, stageOf, entries, &record); err != nil {
		return interviewport.ProgressRecord{}, index{}, err
	}
	restored, err := interview.RestoreProgress(snap)
	if err != nil {
		return interviewport.ProgressRecord{}, index{}, err
	}
	record.Snapshot = restored.Snapshot()
	return record, idx, nil
}

// mapSessions maps voice sessions in open order and returns each session's stage and position.
func mapSessions(rows []SessionModel, record *interviewport.ProgressRecord) (map[string]enum.QuestionStage, map[string]int, error) {
	stageOf := make(map[string]enum.QuestionStage, len(rows))
	order := make(map[string]int, len(rows))
	for i, row := range rows {
		stage := enum.QuestionStage(row.Stage)
		if !stage.Valid() {
			return nil, nil, restoreError("unknown session stage")
		}
		stageOf[row.ID], order[row.ID] = stage, i
		record.Sessions = append(record.Sessions, interviewport.VoiceSession{
			ID: row.ID, AttemptID: row.AttemptID, Stage: stage, ProviderSessionID: textPointer(row.ProviderSessionID),
			ReadinessAt: utcPointer(row.ReadinessAt), ReplacedSessionID: textPointer(row.ReplacedSessionID),
			OpenedAt: utc(row.OpenedAt), ClosedAt: utcPointer(row.ClosedAt),
		})
	}
	return stageOf, order, nil
}

// mapCoverage orders stored items by the pinned question inventory and rejects state it cannot represent.
func mapCoverage(rows []CoverageModel, questions []valueobject.MandatoryQuestion) ([]interview.CoverageItem, error) {
	byQuestion := make(map[string]CoverageModel, len(rows))
	for _, row := range rows {
		byQuestion[row.QuestionID] = row
	}
	if len(rows) != len(questions) || len(byQuestion) != len(rows) {
		return nil, restoreError("coverage does not match the pinned questions")
	}
	items := make([]interview.CoverageItem, 0, len(questions))
	for _, q := range questions {
		row, ok := byQuestion[q.ID()]
		if !ok || (row.ClarificationCount == 1) != (row.ClarificationEntryID != nil) {
			return nil, restoreError("coverage row missing or clarification count disagrees with its entry")
		}
		items = append(items, interview.CoverageItem{
			QuestionID: row.QuestionID, Stage: enum.QuestionStage(row.Stage), State: interview.CoverageState(row.State),
			OmissionReason: interview.OmissionReason(text(row.OmissionReason)), AnswerEntryID: text(row.AnswerEntryID),
			ClarificationEntryID: text(row.ClarificationEntryID), OmissionEntryID: text(row.OmissionEntryID),
		})
	}
	return items, nil
}

// mapEntries maps classified transcript rows to their session's stage in session then ordinal order.
func mapEntries(rows []EntryModel, stageOf map[string]enum.QuestionStage, order map[string]int, idx *index) ([]interview.TranscriptEntry, error) {
	sort.SliceStable(rows, func(i, j int) bool {
		if order[rows[i].SessionID] != order[rows[j].SessionID] {
			return order[rows[i].SessionID] < order[rows[j].SessionID]
		}
		return rows[i].Ordinal < rows[j].Ordinal
	})
	entries := make([]interview.TranscriptEntry, 0, len(rows))
	for _, row := range rows {
		stage, ok := stageOf[row.SessionID]
		if !ok || row.Revision != 1 || row.CitesEntryID != nil || row.SupersedesEntryID != nil {
			return nil, restoreError("transcript row has an unknown session or unrepresentable revision")
		}
		idx.ordinals[row.ID], idx.sessions[row.ID] = row.Ordinal, row.SessionID
		entries = append(entries, interview.TranscriptEntry{ID: row.ID, Stage: stage, Speaker: interview.Speaker(row.Speaker), State: interview.EvidenceState(row.EvidenceState), Body: row.Body})
	}
	return entries, nil
}

// mapHandoffs derives each sealed stage's evidence from the stored stage records and keeps summaries by stage.
func mapHandoffs(rows []HandoffModel, snap *interview.Snapshot, record *interviewport.ProgressRecord) error {
	for i, row := range rows {
		if row.Revision != i+1 {
			return restoreError("handoff revisions are not contiguous")
		}
		h := interview.Handoff{Stage: enum.QuestionStage(row.SourceStage), Target: enum.QuestionStage(text(row.TargetStage)), CommittedAt: utc(row.CreatedAt)}
		stageOfQuestion := make(map[string]enum.QuestionStage, len(snap.Coverage))
		for _, q := range snap.Coverage {
			stageOfQuestion[q.QuestionID] = q.Stage
			if q.Stage == h.Stage {
				h.Coverage = append(h.Coverage, q)
			}
		}
		for _, entry := range snap.Entries {
			if entry.Stage == h.Stage && entry.State == interview.Final {
				h.EntryIDs = append(h.EntryIDs, entry.ID)
			}
		}
		for _, loss := range snap.Losses {
			if stageOfQuestion[loss.QuestionID] == h.Stage {
				h.Losses = append(h.Losses, loss)
			}
		}
		snap.Handoffs = append(snap.Handoffs, h)
		if row.Summary != nil {
			if record.HandoffSummaries == nil {
				record.HandoffSummaries = make(map[enum.QuestionStage]string)
			}
			record.HandoffSummaries[h.Stage] = *row.Summary
		}
	}
	return nil
}

// mapCheckpoints derives each checkpoint's final entry ids from its session ordinal boundary.
func mapCheckpoints(rows []CheckpointModel, stageOf map[string]enum.QuestionStage, entries []EntryModel, record *interviewport.ProgressRecord) error {
	for _, row := range rows {
		stage, ok := stageOf[row.SessionID]
		if !ok {
			return restoreError("checkpoint references an unknown session")
		}
		cp := interviewport.Checkpoint{ID: row.ID, Stage: stage, SessionID: row.SessionID, AttemptID: row.AttemptID, CommittedAt: utc(row.CreatedAt)}
		for _, entry := range entries {
			if entry.SessionID == row.SessionID && entry.Ordinal <= row.ThroughOrdinal && interview.EvidenceState(entry.EvidenceState) == interview.Final {
				cp.EntryIDs = append(cp.EntryIDs, entry.ID)
			}
		}
		record.Checkpoints = append(record.Checkpoints, cp)
	}
	return nil
}
