// Package interviewusecase orchestrates interview rules through lifecycle, store and voice ports.
// Operations are unmounted until the lifecycle and persistence adapters are available.
package interviewusecase

import (
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
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
