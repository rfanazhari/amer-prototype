package candidateinterviewusecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
)

// AcceptNotice stores or finds the acceptance of the current notice version for the resolved candidate session.
type AcceptNotice struct {
	Current  service.CurrentCandidateSession
	Catalog  service.NoticeCatalog
	Tx       candidateinterviewport.ConsentTransaction
	Consents candidateinterviewport.Consents
	Clock    service.Clock
}

// Execute records the acceptance. Input is validated before any port call, an empty catalog writes nothing, and a
// retry or a lost unique-violation race returns the stored row, read back in a new transaction.
func (u AcceptNotice) Execute(ctx context.Context, in AcceptNoticeInput) (*AcceptNoticeResult, error) {
	if !in.Accepted {
		return nil, candidateinterview.ErrInvalidConsent
	}
	accepted, err := candidateinterview.ParseNoticeVersion(in.NoticeVersion)
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
	consent, err := candidateinterview.NewConsent(uuid.NewString(), session.AssignmentID, session.ID, accepted, notice.Version, u.Clock.Now().UTC().Truncate(time.Millisecond))
	if err != nil {
		return nil, err
	}
	snap, created, err := u.store(ctx, session.AssignmentID, session.ID, consent.Snapshot())
	if errors.Is(err, candidateinterviewport.ErrAssignmentUnavailable) {
		return nil, service.ErrCandidateSessionUnbound
	}
	if err != nil {
		return nil, err
	}
	return &AcceptNoticeResult{
		Consent: dto.ConsentView{ID: snap.ID, NoticeVersion: snap.NoticeVersion.String(), AcceptedAt: snap.AcceptedAt},
		Created: created,
	}, nil
}

// store inserts the acceptance unless the assignment already holds that version, and reports whether this call created it.
func (u AcceptNotice) store(ctx context.Context, assignmentID, sessionID string, fresh candidateinterview.ConsentSnapshot) (candidateinterview.ConsentSnapshot, bool, error) {
	var (
		snap    candidateinterview.ConsentSnapshot
		created bool
	)
	err := u.Tx.WithAssignment(ctx, assignmentID, sessionID, func(ctx context.Context) error {
		found, err := u.Consents.Find(ctx, assignmentID, fresh.NoticeVersion)
		if err == nil {
			snap = found
			return nil
		}
		if !errors.Is(err, candidateinterviewport.ErrNotFound) {
			return err
		}
		if err := u.Consents.Insert(ctx, fresh); err != nil {
			return err
		}
		snap, created = fresh, true
		return nil
	})
	if !errors.Is(err, candidateinterviewport.ErrAlreadyExists) {
		return snap, created, err
	}
	// The unique violation aborted the first transaction; read the winner back in a new one.
	err = u.Tx.WithAssignment(ctx, assignmentID, sessionID, func(ctx context.Context) error {
		found, err := u.Consents.Find(ctx, assignmentID, fresh.NoticeVersion)
		if errors.Is(err, candidateinterviewport.ErrNotFound) {
			return service.ErrInterviewDependencyUnavailable
		}
		snap = found
		return err
	})
	return snap, false, err
}

var _ AcceptNoticeOperation = AcceptNotice{}
