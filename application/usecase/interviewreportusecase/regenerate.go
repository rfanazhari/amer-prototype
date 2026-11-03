package interviewreportusecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

// RegenerateCommandKind is the durable command kind of an explicit regeneration.
const RegenerateCommandKind = "regenerate-interview-report/v1"

var requestKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// Regenerate starts a new report generation for a completed interview, or replays an earlier request.
type Regenerate struct {
	Authority    service.InterviewReportAuthority
	Transactions interviewreportport.Transaction
	Assignments  interviewreportport.AssignmentLocker
	Reports      interviewreportport.ReportReader
	Commands     interviewreportport.RegenerationStore
	Evidence     interviewreportport.Evidence
	Clock        service.Clock
	// NewID mints identities in the order report (only when the report is new), generation, job, outbox, command;
	// nil uses random UUIDs.
	NewID func() string
}

var _ RegenerateOperation = Regenerate{}

// Execute authorizes current HR, checks replay before eligibility and withholds the result until commit.
// It never changes the assignment completion marker or the execution.
func (u Regenerate) Execute(ctx context.Context, input RegenerateInput) (*RegenerateResult, error) {
	principal, err := u.Authority.Principal(ctx)
	if err != nil {
		return nil, err
	}
	if err := u.Authority.Authorize(ctx, principal, service.InterviewReportRegenerate, input.AssignmentID); err != nil {
		return nil, err
	}
	if input.AssignmentID == "" || !requestKeyPattern.MatchString(input.IdempotencyKey) {
		return nil, service.ErrReportInvalidInput
	}
	fingerprint, err := regenerateFingerprint(input.AssignmentID)
	if err != nil {
		return nil, err
	}
	identity := interviewreportport.CommandIdentity{Scope: principal.SubjectID, Kind: RegenerateCommandKind, Key: input.IdempotencyKey}
	var result RegenerateResult
	err = u.Transactions.InTransaction(ctx, func(ctx context.Context) error {
		return u.regenerate(ctx, input.AssignmentID, identity, fingerprint, &result)
	})
	if errors.Is(err, interviewreportport.ErrDuplicateCommand) {
		return u.reconcile(ctx, identity, fingerprint, err)
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// regenerate runs the locked part of the command: replay first, then eligibility, then the insert.
func (u Regenerate) regenerate(ctx context.Context, assignmentID string, identity interviewreportport.CommandIdentity, fingerprint string, result *RegenerateResult) error {
	ref, err := u.Assignments.LockAssignment(ctx, assignmentID)
	if err != nil {
		return err
	}
	stored, found, err := u.Commands.FindCommand(ctx, identity)
	if err != nil {
		return err
	}
	if found {
		replayed, err := u.replay(ctx, stored, fingerprint)
		if err != nil {
			return err
		}
		*result = *replayed
		return nil
	}
	if ref.CompletedAt == nil {
		return service.ErrReportNotCompleted
	}
	entries, err := u.Evidence.Finalized(ctx, ref.ExecutionID)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return service.ErrReportEvidenceUnavailable
	}
	now := trustedNow(u.Clock)
	report, creates, err := u.loadReport(ctx, ref, now)
	if err != nil {
		return err
	}
	generation, err := report.StartGeneration(u.mint(), now)
	if err != nil {
		return err
	}
	regeneration := interviewreportport.Regeneration{
		Command:        identity,
		Fingerprint:    fingerprint,
		CreatesReport:  creates,
		Report:         report.Snapshot(),
		WaitDeadlineAt: now.Add(interviewreport.EvidenceWait),
	}
	regeneration.JobID = u.mint()
	regeneration.OutboxID = u.mint()
	regeneration.CommandID = u.mint()
	if err := u.Commands.InsertRegeneration(ctx, regeneration); err != nil {
		return err
	}
	view := dto.NewReportGeneration(generation)
	*result = RegenerateResult{Generation: &view}
	return nil
}

// loadReport restores the stored report, or builds a new one for a completed interview that has none.
// The boolean reports that the report is new.
func (u Regenerate) loadReport(ctx context.Context, ref interviewreportport.AssignmentRef, now time.Time) (*interviewreport.Report, bool, error) {
	snapshot, err := u.Reports.FindByExecution(ctx, ref.ExecutionID)
	if errors.Is(err, interviewreportport.ErrNotFound) {
		report, err := interviewreport.NewReport(u.mint(), ref.ExecutionID, ref.AssignmentID, now)
		return report, true, err
	}
	if err != nil {
		return nil, false, err
	}
	report, err := interviewreport.RestoreReport(snapshot)
	return report, false, err
}

// replay returns the original generation of a stored command whose intent matches, or the conflict.
func (u Regenerate) replay(ctx context.Context, stored interviewreportport.StoredCommand, fingerprint string) (*RegenerateResult, error) {
	if stored.Fingerprint != fingerprint {
		return nil, service.ErrReportIdempotencyConflict
	}
	snapshot, err := u.Reports.FindByExecution(ctx, stored.ExecutionID)
	if err != nil {
		return nil, err
	}
	for _, g := range snapshot.Generations {
		if g.ID == stored.GenerationID {
			view := dto.NewReportGeneration(g)
			return &RegenerateResult{Generation: &view, Replayed: true}, nil
		}
	}
	return nil, interviewreport.ErrGenerationNotFound
}

// reconcile reads the winning command after a unique-identity loss, outside the aborted transaction.
// It returns the original error when the winner cannot be found.
func (u Regenerate) reconcile(ctx context.Context, identity interviewreportport.CommandIdentity, fingerprint string, original error) (*RegenerateResult, error) {
	stored, found, err := u.Commands.FindCommand(ctx, identity)
	if err != nil || !found {
		return nil, original
	}
	return u.replay(ctx, stored, fingerprint)
}

// mint returns the next identity from NewID, or a random UUID when none is injected.
func (u Regenerate) mint() string { return mintID(u.NewID) }

// mintID returns the next identity from newID, or a random UUID when none is injected.
func mintID(newID func() string) string {
	if newID != nil {
		return newID()
	}
	return uuid.NewString()
}

// regenerateFingerprint hashes the compact JSON intent array with the lowercase assignment id.
func regenerateFingerprint(assignmentID string) (string, error) {
	payload, err := json.Marshal([]string{RegenerateCommandKind, strings.ToLower(assignmentID)})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
