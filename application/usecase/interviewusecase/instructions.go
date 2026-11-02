package interviewusecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// defaultDiscussionMode is the only supported compensation mode when a stored policy is absent.
const defaultDiscussionMode = "expectations_only"

// stageInstructions renders the model context for the snapshot's current stage.
// It carries question text, topics, the role name, remaining follow-ups and the discussion mode.
// The ceiling reference, any amount and any verdict request are deliberately never rendered.
func (u Workflow) stageInstructions(ctx context.Context, s interview.Snapshot) (string, error) {
	role, err := u.roleName(ctx, s)
	if err != nil {
		return "", err
	}
	p, err := interview.RestoreProgress(s)
	if err != nil {
		return "", err
	}
	mode := defaultDiscussionMode
	if policy := s.Configuration.CompensationPolicy(); policy != nil && policy.DiscussionMode() != "" {
		mode = policy.DiscussionMode()
	}
	var b strings.Builder
	b.WriteString("You are conducting a structured voice interview.\n")
	if role != "" {
		fmt.Fprintf(&b, "Role: %s\n", role)
	}
	fmt.Fprintf(&b, "Current stage: %s\n", s.CurrentStage)
	b.WriteString("Questions for this stage:\n")
	for _, q := range s.Configuration.MandatoryQuestions() {
		if q.Stage() == s.CurrentStage {
			fmt.Fprintf(&b, "- %s\n", q.Text())
		}
	}
	if topics := s.Configuration.CompetencyTopics(); len(topics) > 0 {
		fmt.Fprintf(&b, "Competency topics: %s\n", strings.Join(topics, ", "))
	}
	fmt.Fprintf(&b, "Follow-ups remaining: %d\n", p.RemainingFollowUps())
	fmt.Fprintf(&b, "Compensation discussion mode: %s\n", mode)
	b.WriteString("You may ask what the candidate expects. Do not state or confirm any internal limit.\n")
	b.WriteString("Record what the candidate says. Do not give any evaluation or outcome of the interview.\n")
	return b.String(), nil
}

// roleName resolves the pinned job-description version to its role, or returns empty when none is pinned.
// A lookup failure, a missing reference or a mismatched identity stops session creation.
func (u Workflow) roleName(ctx context.Context, s interview.Snapshot) (string, error) {
	ref := s.Configuration.JobDescriptionRef()
	if ref == nil {
		return "", nil
	}
	if u.References == nil {
		return "", interview.ErrInvalid
	}
	fact, found, err := u.References.Resolve(ctx, service.AgentReference{Kind: service.ReferenceKindJobDescription, ID: ref.ID(), Version: ref.Version()})
	if err != nil {
		return "", err
	}
	if !found {
		return "", interview.ErrConflict
	}
	if fact.ID != ref.ID() || fact.Version != ref.Version() {
		return "", service.ErrReferenceIdentity
	}
	return fact.Role, nil
}
