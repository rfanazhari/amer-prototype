// Package candidateprogress projects interview runtime progress onto the candidate-safe summary.
package candidateprogress

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// phaseLabels are the neutral candidate-facing names of the four stored stages; they carry no question text.
var phaseLabels = map[enum.QuestionStage]string{
	enum.StageBackground:   "Background",
	enum.StageScreening:    "Screening",
	enum.StageCompensation: "Expectations",
	enum.StageClosing:      "Closing",
}

// Connection tokens published to the candidate.
const (
	connected    = "connected"
	reconnecting = "reconnecting"
	finalizing   = "finalizing"
)

// Reader implements the candidate-safe progress port over the runtime progress store.
type Reader struct {
	Store interviewport.ProgressStore
}

var _ service.InterviewProgress = Reader{}

// Current projects only the phase label and connection state. Transcript, coverage, losses, handoffs and
// controller data are never copied. Absent progress, or a session that is not yet ready, is no summary.
func (r Reader) Current(ctx context.Context, executionID string) (*service.CandidateProgress, error) {
	record, err := r.Store.Find(ctx, executionID)
	if errors.Is(err, interviewport.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	phase, known := phaseLabels[record.Snapshot.CurrentStage]
	if !known {
		return nil, nil
	}
	switch {
	case record.Snapshot.Reconnect.DetectedAt != nil:
		return &service.CandidateProgress{Phase: phase, Connection: reconnecting}, nil
	case !hasReadySession(record.Sessions):
		return nil, nil
	case record.Snapshot.CurrentStage == enum.StageClosing:
		return &service.CandidateProgress{Phase: phase, Connection: finalizing}, nil
	}
	return &service.CandidateProgress{Phase: phase, Connection: connected}, nil
}

// hasReadySession reports whether an open session has both provider identity and browser readiness.
func hasReadySession(sessions []interviewport.VoiceSession) bool {
	for _, s := range sessions {
		if s.ClosedAt == nil && s.ProviderSessionID != nil && s.ReadinessAt != nil {
			return true
		}
	}
	return false
}
