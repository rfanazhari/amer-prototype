package interview

import "github.com/rfanazhari/amer-prototype/domain/enum"

// RestoreProgress restores stored policy without publication reassessment.
// Structural corruption is a technical error rather than invalid candidate input.
func RestoreProgress(s Snapshot) (*Progress, error) {
	p := &Progress{state: s.detach()}
	if !p.validHeader() || !p.validEntries() || !p.validCoverage() || !p.validFollowUps() || !p.validLosses() || !p.validHandoffs() || !p.validReconnect() {
		return nil, ErrRestore
	}
	return p, nil
}

// validHeader checks identity, stage, stored timers and policy without current publication rules.
func (p *Progress) validHeader() bool {
	s := p.state
	if s.ExecutionID == "" || s.RevisionID == "" || !s.CurrentStage.Valid() || s.FollowUpLimit < 1 || s.RunningAt.IsZero() || s.StageActivatedAt.IsZero() || s.StageActivatedAt.Before(s.RunningAt) || p.HardDeadlineReached(s.StageActivatedAt) {
		return false
	}
	policy := s.Configuration.CompensationPolicy()
	if policy == nil || policy.DiscussionMode() != "expectations_only" {
		return false
	}
	for _, stage := range stages() {
		if stageSeconds(s.Configuration, stage) <= 0 {
			return false
		}
	}
	return true
}

// validEntries checks classifications and unique local identities in reached stages.
func (p *Progress) validEntries() bool {
	seen := make(map[string]bool, len(p.state.Entries))
	for _, entry := range p.state.Entries {
		if !entry.valid() || seen[entry.ID] || stageIndex(entry.Stage) > stageIndex(p.state.CurrentStage) {
			return false
		}
		seen[entry.ID] = true
	}
	return true
}

// validCoverage matches the pinned question inventory and checks all state-specific citations.
func (p *Progress) validCoverage() bool {
	questions := p.state.Configuration.MandatoryQuestions()
	if len(questions) != len(p.state.Coverage) {
		return false
	}
	for i, q := range p.state.Coverage {
		if q.QuestionID != questions[i].ID() || q.Stage != questions[i].Stage() || !p.validItem(q) {
			return false
		}
		if stageIndex(q.Stage) > stageIndex(p.state.CurrentStage) && q.State != Pending {
			return false
		}
	}
	return true
}

// validItem checks terminal fields and evidence independently of publication constants.
func (p *Progress) validItem(q CoverageItem) bool {
	if q.ClarificationEntryID != "" && !p.citation(q.ClarificationEntryID, q.Stage, Model) {
		return false
	}
	switch q.State {
	case Pending:
		return q.ClarificationEntryID == "" && q.AnswerEntryID == "" && q.OmissionReason == "" && q.OmissionEntryID == ""
	case Asked:
		return q.AnswerEntryID == "" && q.OmissionReason == "" && q.OmissionEntryID == ""
	case Answered:
		return p.citation(q.AnswerEntryID, q.Stage, Candidate) && q.OmissionReason == "" && q.OmissionEntryID == ""
	case Omitted:
		if q.AnswerEntryID != "" {
			return false
		}
		switch q.OmissionReason {
		case CandidateSkip:
			return q.ClarificationEntryID != "" && p.citation(q.OmissionEntryID, q.Stage, Candidate)
		case NoAnswerAfterClarification:
			return q.ClarificationEntryID != "" && q.OmissionEntryID == ""
		case SoftBudgetExpired:
			return q.OmissionEntryID == ""
		}
	}
	return false
}

// validFollowUps counts stored citations by stage using the stored cap, not the creation cap.
func (p *Progress) validFollowUps() bool {
	counts := make(map[enum.QuestionStage]int)
	for _, id := range p.state.FollowUps {
		found := false
		for _, entry := range p.state.Entries {
			if entry.ID == id && entry.Speaker == Candidate && entry.State == Final {
				counts[entry.Stage]++
				if counts[entry.Stage] > p.state.FollowUpLimit {
					return false
				}
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// coverageByID finds a stored question regardless of whether its stage is sealed.
func (p *Progress) coverageByID(id string) (CoverageItem, bool) {
	for _, q := range p.state.Coverage {
		if q.QuestionID == id {
			return q, true
		}
	}
	return CoverageItem{}, false
}

// validLosses rejects duplicate gaps, unknown questions and contradictions with terminal coverage.
func (p *Progress) validLosses() bool {
	seen := make(map[string]bool, len(p.state.Losses))
	for _, loss := range p.state.Losses {
		q, ok := p.coverageByID(loss.QuestionID)
		if !ok || loss.ID == "" || seen[loss.ID] || stageIndex(q.Stage) > stageIndex(p.state.CurrentStage) {
			return false
		}
		if (loss.Resolved && q.State != Answered) || (!loss.Resolved && (q.State == Answered || q.State == Omitted)) {
			return false
		}
		seen[loss.ID] = true
	}
	return true
}

// validHandoffs verifies fixed ordering, sealed evidence and source-before-target timestamps.
func (p *Progress) validHandoffs() bool {
	s := p.state
	current := stageIndex(s.CurrentStage)
	if len(s.Handoffs) < current || len(s.Handoffs) > current+1 {
		return false
	}
	for i, h := range s.Handoffs {
		if h.Stage != stages()[i] || h.Target != nextStage(h.Stage) || h.CommittedAt.IsZero() || h.CommittedAt.Before(s.RunningAt) || p.HardDeadlineReached(h.CommittedAt) || !p.validHandoffRecord(h) {
			return false
		}
		if i > 0 && h.CommittedAt.Before(s.Handoffs[i-1].CommittedAt) {
			return false
		}
		if i == current && h.CommittedAt.Before(s.StageActivatedAt) {
			return false
		}
		if i < current && s.StageActivatedAt.Before(h.CommittedAt) {
			return false
		}
	}
	return true
}

// validHandoffRecord requires exactly the source coverage, finalized transcript ids and gaps.
func (p *Progress) validHandoffRecord(h Handoff) bool {
	coverage := make([]CoverageItem, 0)
	losses := make([]TechnicalLoss, 0)
	entries := make(map[string]bool)
	for _, q := range p.state.Coverage {
		if q.Stage == h.Stage {
			coverage = append(coverage, q)
		}
	}
	if len(coverage) != len(h.Coverage) {
		return false
	}
	for i, q := range coverage {
		if q != h.Coverage[i] {
			return false
		}
	}
	for _, loss := range p.state.Losses {
		q, _ := p.coverageByID(loss.QuestionID)
		if q.Stage == h.Stage {
			losses = append(losses, loss)
		}
	}
	if len(losses) != len(h.Losses) {
		return false
	}
	for i, loss := range losses {
		if loss != h.Losses[i] {
			return false
		}
	}
	for _, entry := range p.state.Entries {
		if entry.Stage == h.Stage && entry.State == Final {
			entries[entry.ID] = true
		}
	}
	if len(entries) != len(h.EntryIDs) {
		return false
	}
	for _, id := range h.EntryIDs {
		if !entries[id] {
			return false
		}
		delete(entries, id)
	}
	return true
}

// validReconnect checks persisted counter ranges and the backend detection instant.
func (p *Progress) validReconnect() bool {
	r := p.state.Reconnect
	if r.Episodes < 0 || r.Episodes > 2 || r.Attempts < 0 || r.Attempts > 2 {
		return false
	}
	if r.Episodes == 0 {
		return r.Attempts == 0 && r.DetectedAt == nil
	}
	return r.DetectedAt == nil || (!r.DetectedAt.IsZero() && !r.DetectedAt.Before(p.state.RunningAt) && !p.HardDeadlineReached(*r.DetectedAt))
}
