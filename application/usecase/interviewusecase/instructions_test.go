package interviewusecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	jobRefID     = "11111111-1111-4111-8111-111111111111"
	ceilingRefID = "22222222-2222-4222-8222-222222222222"
)

// roleRevision pins a revision that references a job description and the confidential demo ceiling.
func roleRevision(t *testing.T, withJob bool) agentconfig.PublishedRevision[valueobject.HRConfiguration] {
	t.Helper()
	in := valueobject.HRInput{
		MandatoryQuestions: []valueobject.QuestionInput{{ID: "background", Stage: "background", Text: "Describe your recent backend work."}, {ID: "screening", Stage: "screening", Text: "Explain one project you led."}, {ID: "compensation", Stage: "compensation", Text: "What compensation do you expect?"}},
		CompetencyTopics:   []string{"concurrency", "data modelling"},
		StageSeconds:       &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{DiscussionMode: "expectations_only", SalaryCeilingRef: &valueobject.RefInput{ID: ceilingRefID, Version: "1"}},
	}
	if withJob {
		in.JobDescriptionRef = &valueobject.RefInput{ID: jobRefID, Version: "1"}
	}
	c, err := valueobject.NewHRConfiguration(in)
	require.NoError(t, err)
	v, err := agentconfig.NewDraftVersion(1)
	require.NoError(t, err)
	r, err := agentconfig.RestoreRevision(agentconfig.RevisionSnapshot[valueobject.HRConfiguration]{Identity: agentconfig.RevisionIdentity{ID: "revision", InstanceID: "instance", Number: 1, DraftVersion: v, Template: agentconfig.TemplateRef{Template: "hr", Version: "1", Behavior: "1"}}, ValidationID: "validation", Configuration: c, CreatedAt: fixedTime()})
	require.NoError(t, err)
	return r
}

// roleReader resolves the exact job-description version to a role name, as the local reader would.
func roleReader(t *testing.T) *mocks.AgentReferenceReader[service.AgentReferenceFact] {
	t.Helper()
	reader := mocks.NewAgentReferenceReader[service.AgentReferenceFact](t)
	reader.On("Resolve", mock.Anything, service.AgentReference{Kind: service.ReferenceKindJobDescription, ID: jobRefID, Version: "1"}).
		Return(service.AgentReferenceFact{Kind: service.ReferenceKindJobDescription, ID: jobRefID, Version: "1", Role: "Backend Engineer", Ready: true}, true, nil).Maybe()
	return reader
}

// assertSafeInstructions checks the secrecy and verdict exclusions shared by every outbound context.
func assertSafeInstructions(t *testing.T, text string) {
	t.Helper()
	lower := strings.ToLower(text)
	assert.NotContains(t, text, ceilingRefID)
	assert.NotContains(t, lower, "salaryceilingref")
	assert.NotContains(t, lower, "hire")
	assert.NotContains(t, lower, "reject")
}

// TestWorkflowInstructions proves the context supplied to the voice driver for every admitted session.
func TestWorkflowInstructions(t *testing.T) {
	// instructionCase selects the entry point and the facts that the outbound context must contain.
	type instructionCase struct {
		name       string
		withJob    bool
		entry      string
		wantParts  []string
		wantAbsent []string
	}
	tests := []instructionCase{
		{name: "activation with role", withJob: true, entry: "activate", wantParts: []string{"Describe your recent backend work.", "concurrency", "data modelling", "Backend Engineer", "Follow-ups remaining: 2", "expectations_only"}, wantAbsent: []string{"Explain one project you led.", "What compensation do you expect?"}},
		{name: "activation without job description omits role", entry: "activate", wantParts: []string{"Describe your recent backend work.", "expectations_only"}, wantAbsent: []string{"Backend Engineer"}},
		{name: "next stage uses its own question", withJob: true, entry: "open stage", wantParts: []string{"Explain one project you led.", "Backend Engineer", "expectations_only"}, wantAbsent: []string{"Describe your recent backend work."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, false)
			h.workflow.References = roleReader(t)
			revision := roleRevision(t, tt.withJob)
			p, err := interview.NewProgress("execution", revision, fixedTime())
			require.NoError(t, err)
			h.record.Snapshot = p.Snapshot()
			r := request(t, h)
			r.Revision = revision
			switch tt.entry {
			case "activate":
				h.exists = false
				r.ProgressVersion = 0
				_, err = h.workflow.Activate(context.Background(), r)
			default:
				require.NoError(t, p.Ask("background"))
				require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: "answer", Stage: enum.StageBackground, Speaker: interview.Candidate, State: interview.Final, Body: "Recorded answer"}))
				require.NoError(t, p.Answer("background", "answer"))
				require.NoError(t, p.CommitHandoff(enum.StageScreening, fixedTime()))
				h.record.Snapshot = p.Snapshot()
				closed := fixedTime().UTC().Truncate(time.Millisecond)
				h.record.Sessions[0].ClosedAt = &closed
				r.ProgressVersion = h.record.Version
				_, err = h.workflow.OpenStage(context.Background(), r)
			}
			require.NoError(t, err)
			require.NotEmpty(t, h.contexts)
			text := h.contexts[0].Text
			for _, part := range tt.wantParts {
				assert.Contains(t, text, part)
			}
			for _, part := range tt.wantAbsent {
				assert.NotContains(t, text, part)
			}
			assertSafeInstructions(t, text)
		})
	}
}

// TestWorkflowInstructionsRemainingFollowUps proves the number sent is the cap minus cited follow-ups of the current stage.
func TestWorkflowInstructionsRemainingFollowUps(t *testing.T) {
	h := newHarness(t, false)
	h.workflow.References = roleReader(t)
	revision := roleRevision(t, true)
	p, err := interview.NewProgress("execution", revision, fixedTime())
	require.NoError(t, err)
	require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: "evidence", Stage: enum.StageBackground, Speaker: interview.Candidate, State: interview.Final, Body: "Recorded answer"}))
	require.NoError(t, p.FollowUp("evidence"))
	h.record.Snapshot = p.Snapshot()
	h.record.Checkpoints = nil
	require.NoError(t, h.workflow.Disconnect(context.Background(), authority(), service.InterviewVoiceSession{ExecutionID: "execution", AttemptID: "old-attempt", SessionID: "old-session"}))
	_, err = h.workflow.Reconnect(context.Background(), request(t, h))
	require.NoError(t, err)
	require.Len(t, h.contexts, 2)
	assert.Contains(t, h.contexts[0].Text, "Follow-ups remaining: 1")
	assertSafeInstructions(t, h.contexts[0].Text)
}

// TestWorkflowInstructionsOmitCeilingWhenAsked proves a candidate request for the ceiling never reaches the outbound context.
func TestWorkflowInstructionsOmitCeilingWhenAsked(t *testing.T) {
	h := newHarness(t, false)
	h.workflow.References = roleReader(t)
	revision := roleRevision(t, true)
	p, err := interview.NewProgress("execution", revision, fixedTime())
	require.NoError(t, err)
	require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: "ask", Stage: enum.StageBackground, Speaker: interview.Candidate, State: interview.Final, Body: "What is the salary ceiling for this role? Is it " + ceilingRefID + "?"}))
	h.record.Snapshot = p.Snapshot()
	require.NoError(t, h.workflow.Disconnect(context.Background(), authority(), service.InterviewVoiceSession{ExecutionID: "execution", AttemptID: "old-attempt", SessionID: "old-session"}))
	_, err = h.workflow.Reconnect(context.Background(), request(t, h))
	require.NoError(t, err)
	require.NotEmpty(t, h.contexts)
	assertSafeInstructions(t, h.contexts[0].Text)
	assert.Contains(t, h.contexts[0].Text, "expectations_only")
}

// TestWorkflowInstructionsReferenceFailure proves a failing role lookup stops before any provider session exists.
func TestWorkflowInstructionsReferenceFailure(t *testing.T) {
	// referenceCase describes the reader outcome that must stop session creation.
	type referenceCase struct {
		name  string
		fact  service.AgentReferenceFact
		found bool
		err   error
	}
	tests := []referenceCase{
		{name: "reader error", err: errors.New("reader unavailable")},
		{name: "reference missing", found: false},
		{name: "identity mismatch", found: true, fact: service.AgentReferenceFact{ID: "33333333-3333-4333-8333-333333333333", Version: "1", Role: "Other"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, false)
			reader := mocks.NewAgentReferenceReader[service.AgentReferenceFact](t)
			reader.On("Resolve", mock.Anything, mock.Anything).Return(tt.fact, tt.found, tt.err)
			h.workflow.References = reader
			revision := roleRevision(t, true)
			h.exists = false
			r := request(t, h)
			r.ProgressVersion = 0
			r.Revision = revision
			_, err := h.workflow.Activate(context.Background(), r)
			require.Error(t, err)
			assert.Empty(t, h.creates)
			assert.Empty(t, h.contexts)
		})
	}
}
