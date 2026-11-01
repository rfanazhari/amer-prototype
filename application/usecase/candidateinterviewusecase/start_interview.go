package candidateinterviewusecase

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
)

// idempotencyKeyPattern is the local Idempotency-Key grammar; it matches the publication key rule without importing it.
var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// StartInterview checks consent and coverage, then asks lifecycle to commit or find the execution.
type StartInterview struct {
	Current     service.CurrentCandidateSession
	Catalog     service.NoticeCatalog
	Consents    candidateinterviewport.Consents
	Controllers candidateinterviewport.Controllers
	Lifecycle   service.ExecutionLifecycle
	Clock       service.Clock
}

// Execute validates input, then requires stored acceptance of the current notice and a session that outlasts the
// interview budget before the single lifecycle call. It writes no consent and claims the controller for the calling tab.
func (u StartInterview) Execute(ctx context.Context, in StartInterviewInput) (*StartInterviewResult, error) {
	if !idempotencyKeyPattern.MatchString(in.IdempotencyKey) {
		return nil, ErrInvalidIdempotencyKey
	}
	tab, err := candidateinterview.ParseControllerID(in.ControllerID)
	if err != nil {
		return nil, err
	}
	session, err := u.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	notice, ok := u.Catalog.Current(ctx)
	if !ok {
		return nil, service.ErrInterviewDependencyUnavailable
	}
	consent, err := u.Consents.FindByID(ctx, session.AssignmentID, in.ConsentID)
	if errors.Is(err, candidateinterviewport.ErrNotFound) || (err == nil && consent.NoticeVersion != notice.Version) {
		return nil, ErrConsentRequired
	}
	if err != nil {
		return nil, err
	}
	now := u.Clock.Now().UTC().Truncate(time.Millisecond)
	if !candidateinterview.SessionCovers(session.ExpiresAt, now.Add(candidateinterview.InterviewBudget)) {
		return nil, service.ErrInterviewCoverage
	}
	committed, err := u.Lifecycle.CommitOrFind(ctx, service.CommitOrFindRequest{
		AssignmentID: session.AssignmentID, IdempotencyKey: in.IdempotencyKey, ConsentID: in.ConsentID,
	})
	if err != nil {
		return nil, err
	}
	isController, version, err := u.claim(ctx, committed.Execution, tab, now)
	if err != nil {
		return nil, err
	}
	return &StartInterviewResult{Execution: executionView(committed.Execution, isController, version), Created: committed.Created}, nil
}

// claim makes tab the controller of a new interview, or reports who controls an existing one.
func (u StartInterview) claim(ctx context.Context, facts service.ExecutionFacts, tab candidateinterview.ControllerID, now time.Time) (bool, int64, error) {
	claimed, err := candidateinterview.ClaimController(facts.AssignmentID, facts.ID, tab, now)
	if err != nil {
		return false, 0, err
	}
	err = u.Controllers.Claim(ctx, claimed.Snapshot())
	if err == nil {
		return true, claimed.Snapshot().Version, nil
	}
	if !errors.Is(err, candidateinterviewport.ErrAlreadyExists) {
		return false, 0, err
	}
	existing, err := u.Controllers.Find(ctx, facts.AssignmentID)
	if err != nil {
		return false, 0, err
	}
	return existing.ControllerID == tab, existing.Version, nil
}

var _ StartInterviewOperation = StartInterview{}
