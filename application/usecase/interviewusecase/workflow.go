// Package interviewusecase orchestrates interview rules through lifecycle, store and voice ports.
// Production composition supplies the durable lifecycle and persistence adapters.
package interviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// Workflow coordinates one execution without owning execution state or provider classification.
type Workflow struct {
	// ProviderReady rejects missing server credentials before durable admission.
	ProviderReady func() bool
	// Transactions joins admission, progress and command reservation before external setup.
	Transactions executionport.Transactor
	// Commands stores HTTP reservations when a request carries a scoped command identity.
	Commands  interviewport.SessionCommands
	Lifecycle service.InterviewLifecycle
	Store     interviewport.ProgressStore
	Driver    service.InterviewDriver
	Clock     service.Clock
	Summary   service.InterviewSummarizer
	// References resolves the role name for a pinned job-description version.
	References service.AgentReferenceReader[service.AgentReferenceFact]
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
	// Command is present for candidate HTTP setup and reserved with its session shell.
	Command *interviewport.SessionCommand
	// Check revalidates the candidate controller under the execution lock.
	Check           func(context.Context) error
	Authority       service.InterviewAttempt
	ProgressVersion int64
	SessionID       string
	IdempotencyKey  string
	SDP             string
	Revision        agentconfig.PublishedRevision[valueobject.HRConfiguration]
}
