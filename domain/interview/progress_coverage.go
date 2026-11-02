package interview

import (
	"strings"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// question finds a current, unsealed question that operations may mutate.
func (p *Progress) question(id string) *CoverageItem {
	if !p.open() {
		return nil
	}
	for i := range p.state.Coverage {
		q := &p.state.Coverage[i]
		if q.QuestionID == id && q.Stage == p.state.CurrentStage {
			return q
		}
	}
	return nil
}

// citation checks finalized speech in the specified stage and speaker namespace.
func (p *Progress) citation(id string, stage enum.QuestionStage, speaker Speaker) bool {
	if id == "" {
		return false
	}
	for _, entry := range p.state.Entries {
		if entry.ID == id {
			return entry.Stage == stage && entry.State == Final && entry.Speaker == speaker
		}
	}
	return false
}

// unresolved reports a technical gap that cannot be turned into an omission.
func (p *Progress) unresolved(questionID string) bool {
	for _, loss := range p.state.Losses {
		if loss.QuestionID == questionID && !loss.Resolved {
			return true
		}
	}
	return false
}

// Ask moves a pending question into asked coverage.
func (p *Progress) Ask(questionID string) error {
	q := p.question(questionID)
	if q == nil || q.State != Pending {
		return ErrConflict
	}
	q.State = Asked
	return nil
}

// RecordEntry accepts classified transcript evidence without interpreting provider events.
func (p *Progress) RecordEntry(entry TranscriptEntry) error {
	if !p.open() || entry.Stage != p.state.CurrentStage || !entry.valid() {
		return ErrEvidence
	}
	for _, stored := range p.state.Entries {
		if stored.ID == entry.ID {
			return ErrConflict
		}
	}
	p.state.Entries = append(p.state.Entries, entry)
	return nil
}

// valid checks structural speech classification without assuming any provider event map.
func (e TranscriptEntry) valid() bool {
	return e.ID != "" && e.Stage.Valid() && (e.Speaker == Candidate || e.Speaker == Model) && (e.State == Fragment || e.State == Final) && strings.TrimSpace(e.Body) != ""
}

// Clarify records exactly one finalized model clarification for an asked question.
func (p *Progress) Clarify(questionID, entryID string) error {
	q := p.question(questionID)
	if q == nil || q.State != Asked || q.ClarificationEntryID != "" {
		return ErrConflict
	}
	if !p.citation(entryID, q.Stage, Model) {
		return ErrEvidence
	}
	q.ClarificationEntryID = entryID
	return nil
}

// Answer records a final candidate citation and resolves that question's technical gaps.
func (p *Progress) Answer(questionID, entryID string) error {
	q := p.question(questionID)
	if q == nil || q.State != Asked {
		return ErrConflict
	}
	if !p.citation(entryID, q.Stage, Candidate) {
		return ErrEvidence
	}
	q.State = Answered
	q.AnswerEntryID = entryID
	for i := range p.state.Losses {
		if p.state.Losses[i].QuestionID == questionID {
			p.state.Losses[i].Resolved = true
		}
	}
	return nil
}

// Omit records one of three approved reasons when its prerequisites hold.
func (p *Progress) Omit(questionID string, reason OmissionReason, entryID string, now time.Time) error {
	q := p.question(questionID)
	if q == nil || !p.activeAt(trustedTime(now)) || (q.State != Pending && q.State != Asked) || p.unresolved(questionID) {
		return ErrConflict
	}
	switch reason {
	case CandidateSkip:
		if q.State != Asked || q.ClarificationEntryID == "" || !p.citation(entryID, q.Stage, Candidate) {
			return ErrEvidence
		}
	case NoAnswerAfterClarification:
		if q.State != Asked || q.ClarificationEntryID == "" || entryID != "" {
			return ErrEvidence
		}
	case SoftBudgetExpired:
		if !p.SoftBudgetReached(now) || entryID != "" {
			return ErrBudget
		}
	default:
		return ErrInvalid
	}
	q.State = Omitted
	q.OmissionReason = reason
	q.OmissionEntryID = entryID
	return nil
}

// RecordTechnicalLoss preserves open coverage and blocks a stage proposal.
func (p *Progress) RecordTechnicalLoss(id, questionID string) error {
	q := p.question(questionID)
	if id == "" || q == nil || (q.State != Pending && q.State != Asked) {
		return ErrConflict
	}
	for _, loss := range p.state.Losses {
		if loss.ID == id {
			return ErrConflict
		}
	}
	p.state.Losses = append(p.state.Losses, TechnicalLoss{ID: id, QuestionID: questionID})
	return nil
}

// FollowUp consumes the stored stage cap only with a final candidate citation.
func (p *Progress) FollowUp(entryID string) error {
	if !p.open() || !p.citation(entryID, p.state.CurrentStage, Candidate) {
		return ErrEvidence
	}
	count := 0
	for _, id := range p.state.FollowUps {
		if p.citation(id, p.state.CurrentStage, Candidate) {
			count++
		}
	}
	if count >= p.state.FollowUpLimit {
		return ErrBudget
	}
	p.state.FollowUps = append(p.state.FollowUps, entryID)
	return nil
}

// RemainingFollowUps returns how many cited follow-ups the current stage may still use.
func (p *Progress) RemainingFollowUps() int { return 0 }
