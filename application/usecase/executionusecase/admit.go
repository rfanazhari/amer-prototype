package executionusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
)

// Admit commits a CREATED execution for a caller-resolved binding without an assignment or pin.
type Admit struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
	// NewID mints the execution identity; nil uses a random UUID.
	NewID func() string
}

var _ AdmitOperation = Admit{}

// errAdmitReplay aborts the admit transaction so the insert of a duplicate execution rolls back.
var errAdmitReplay = errors.New("admit replay of an existing execution")

// Execute admits a new execution or returns the existing one for the scope and key.
// Shape errors return before any transaction is opened.
func (u Admit) Execute(ctx context.Context, input AdmitInput) (*dto.ExecutionView, error) {
	if !validToken(input.Scope) || !validToken(input.RequestKey) || !validToken(input.ActorRef) ||
		!validUUID(input.InstanceID) || !validUUID(input.RevisionID) || !validRefs(input.InputRefs) {
		return nil, service.ErrExecutionInvalidInput
	}
	var (
		view     *dto.ExecutionView
		storedID string
	)
	err := u.Transactions.WithTransaction(ctx, func(ctx context.Context) error {
		_, snap, fact, err := execution.NewExecution(mintID(u.NewID),
			execution.Binding{InstanceID: input.InstanceID, RevisionID: input.RevisionID}, trustedNow(u.Clock))
		if err != nil {
			return err
		}
		fact.ActorRef = input.ActorRef
		if err := u.Executions.Insert(ctx, snap, fact); err != nil {
			return err
		}
		fingerprint := intentFingerprint(input.Scope, input.InstanceID, input.RevisionID, input.InputRefs)
		id, err := u.Commands.FindOrInsert(ctx, input.Scope, startCommandKind, input.RequestKey, fingerprint, snap.ID)
		if err != nil {
			return err
		}
		if id != snap.ID {
			storedID = id
			return errAdmitReplay
		}
		view = dto.NewExecutionView(snap, "")
		return nil
	})
	if errors.Is(err, errAdmitReplay) {
		stored, findErr := u.Executions.Find(ctx, storedID)
		if findErr != nil {
			return nil, mapStoreError(findErr)
		}
		return dto.NewExecutionView(stored, ""), nil
	}
	if err != nil {
		return nil, mapStoreError(err)
	}
	return view, nil
}
