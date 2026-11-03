package executionusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
)

// FencedWrite admits a sibling write under the same execution transaction and lock.
type FencedWrite struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Attempts     executionport.AttemptLedger
	Clock        service.Clock
}

// Within validates current authority before invoking a transaction-joining write.
func (u FencedWrite) Within(ctx context.Context, id, attemptID string, version int64, maximum time.Duration, write func(context.Context) error) error {
	return u.Transactions.WithTransaction(ctx, func(tx context.Context) error {
		return u.Executions.Update(tx, id, func(snap *execution.Snapshot) (*execution.TransitionFact, error) {
			if terminalState(snap.State) || snap.State != execution.StateRunning {
				return nil, execution.ErrTerminalConflict
			}
			if snap.StateVersion != version {
				return nil, execution.ErrStaleVersion
			}
			a, err := u.Attempts.Find(tx, id, attemptID)
			if err != nil {
				return nil, err
			}
			if a.Generation != snap.CurrentGeneration {
				return nil, execution.ErrStaleAttempt
			}
			if maximum > 0 && (snap.ActivatedAt == nil || !trustedNow(u.Clock).Before(snap.ActivatedAt.Add(maximum))) {
				return nil, execution.ErrStaleAttempt
			}
			return nil, write(tx)
		})
	})
}
