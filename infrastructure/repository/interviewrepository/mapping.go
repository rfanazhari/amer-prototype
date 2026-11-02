package interviewrepository

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// coverageModel maps one question's mutable state; clarification evidence is never representable, so its count is zero.
func coverageModel(executionID, id string, q interview.CoverageItem, at time.Time) CoverageModel {
	model := CoverageModel{ID: id, ExecutionID: executionID, QuestionID: q.QuestionID, Stage: q.Stage.String(), State: string(q.State), UpdatedAt: at}
	if q.OmissionReason != "" {
		reason := string(q.OmissionReason)
		model.OmissionReason = &reason
	}
	if q.AnswerEntryID != "" {
		entry := q.AnswerEntryID
		model.AnswerEntryID = &entry
	}
	return model
}

// sessionByID finds a session of the record by its local id.
func sessionByID(sessions []interviewport.VoiceSession, id string) (interviewport.VoiceSession, bool) {
	for _, v := range sessions {
		if v.ID == id {
			return v, true
		}
	}
	return interviewport.VoiceSession{}, false
}

// equalText compares two nullable strings by value.
func equalText(a, b *string) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

// equalTime compares two nullable instants by value.
func equalTime(a, b *time.Time) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Equal(*b))
}

// cloneRecord detaches slices, maps and pointers so a callback cannot alias the loaded baseline.
func cloneRecord(rec interviewport.ProgressRecord) interviewport.ProgressRecord {
	out := rec
	out.ControllerID = textPointer(rec.ControllerID)
	s := rec.Snapshot
	s.Configuration = s.Configuration.Clone()
	s.Coverage = append([]interview.CoverageItem(nil), s.Coverage...)
	s.Entries = append([]interview.TranscriptEntry(nil), s.Entries...)
	s.FollowUps = append([]string(nil), s.FollowUps...)
	s.Losses = append([]interview.TechnicalLoss(nil), s.Losses...)
	s.Handoffs = append([]interview.Handoff(nil), s.Handoffs...)
	for i := range s.Handoffs {
		h := &s.Handoffs[i]
		h.Coverage = append([]interview.CoverageItem(nil), h.Coverage...)
		h.EntryIDs = append([]string(nil), h.EntryIDs...)
		h.Losses = append([]interview.TechnicalLoss(nil), h.Losses...)
	}
	s.Reconnect.DetectedAt = utcPointer(s.Reconnect.DetectedAt)
	out.Snapshot = s
	out.Sessions = make([]interviewport.VoiceSession, len(rec.Sessions))
	for i, v := range rec.Sessions {
		v.ProviderSessionID, v.ReadinessAt = textPointer(v.ProviderSessionID), utcPointer(v.ReadinessAt)
		v.ReplacedSessionID, v.ClosedAt = textPointer(v.ReplacedSessionID), utcPointer(v.ClosedAt)
		out.Sessions[i] = v
	}
	out.Checkpoints = make([]interviewport.Checkpoint, len(rec.Checkpoints))
	for i, cp := range rec.Checkpoints {
		cp.EntryIDs = append([]string(nil), cp.EntryIDs...)
		out.Checkpoints[i] = cp
	}
	if rec.HandoffSummaries != nil {
		out.HandoffSummaries = make(map[enum.QuestionStage]string, len(rec.HandoffSummaries))
		for stage, summary := range rec.HandoffSummaries {
			out.HandoffSummaries[stage] = summary
		}
	}
	return out
}
