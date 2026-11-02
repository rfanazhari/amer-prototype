// Package interviewusecase orchestrates interview rules through lifecycle, store and voice ports.
// Operations are unmounted until the lifecycle and persistence adapters are available.
package interviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// Workflow coordinates one execution without owning execution state or provider classification.
type Workflow struct {
	Lifecycle service.InterviewLifecycle
	Store     interviewport.ProgressStore
	Driver    service.InterviewDriver
	Clock     service.Clock
	Summary   service.InterviewSummarizer
}

// Proposal carries lifecycle authority and the fixed successor proposed by the model.
type Proposal struct {
	Authority       service.InterviewAttempt
	ProgressVersion int64
	Target          enum.QuestionStage
}

// SessionRequest supplies local identities and the offer for one admitted session.
// Revision is the caller's pinned immutable revision, used only for initial activation.
type SessionRequest struct {
	Authority       service.InterviewAttempt
	ProgressVersion int64
	SessionID       string
	IdempotencyKey  string
	SDP             string
	Revision        agentconfig.PublishedRevision[valueobject.HRConfiguration]
}

// Propose authorizes and commits structured evidence before any later session creation.
func (u Workflow) Propose(context.Context, Proposal) error { return interview.ErrConflict }

// Activate requests the lifecycle's initial activation budget before opening a session.
func (u Workflow) Activate(context.Context, SessionRequest) (service.InterviewVoiceCreated, error) {
	return service.InterviewVoiceCreated{}, interview.ErrConflict
}

// OpenStage opens only the successor of a committed handoff, without inventing a retry budget.
func (u Workflow) OpenStage(context.Context, SessionRequest) (service.InterviewVoiceCreated, error) {
	return service.InterviewVoiceCreated{}, interview.ErrConflict
}

// Disconnect records backend detection for the current session without classifying speech.
func (u Workflow) Disconnect(context.Context, service.InterviewAttempt, service.InterviewVoiceSession) error {
	return interview.ErrConflict
}

// Reconnect replaces a session on the same execution using committed recovery state.
func (u Workflow) Reconnect(context.Context, SessionRequest) (service.InterviewVoiceCreated, error) {
	return service.InterviewVoiceCreated{}, interview.ErrConflict
}
