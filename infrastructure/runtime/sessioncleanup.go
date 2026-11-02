package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
)

// PruneAdministrativeSessions removes old administrative sessions with the runtime database role and returns the number of rows removed; it logs nothing and exposes no credential.
func PruneAdministrativeSessions(ctx context.Context, databaseURL string) (int64, error) {
	db, resources, err := openDatabase(ctx, databaseURL, func() {})
	if err != nil {
		return 0, err
	}
	removed, runErr := (accessusecase.PruneSessions{Retention: accessrepository.Store{DB: db}, Clock: clock.System{}}).Execute(ctx)
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ShutdownBudget)
	defer cancel()
	if err := resources.close(closeCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("close database: %w", err))
	}
	return removed, runErr
}
