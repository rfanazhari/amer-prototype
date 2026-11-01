package accessusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Logout persists current-session revocation before cookie clearing.
type Logout struct {
	Current  service.CurrentAdministrativeSession
	Sessions accessport.Sessions
	Clock    service.Clock
}

// Execute revokes the current administrative session or reports commit uncertainty.
func (u Logout) Execute(ctx context.Context) error {
	session, account, err := u.Current.Resolve(ctx)
	if err != nil {
		return err
	}
	now := trustedNow(u.Clock)
	if err := authorizeCurrent(session, account, now); err != nil {
		return err
	}
	restored, err := restoreSession(session)
	if err != nil {
		return err
	}
	if err := restored.Revoke(now); err != nil {
		return err
	}
	snapshot := restored.Snapshot()
	return u.Sessions.UpdateRevocation(ctx, snapshot.ID, snapshot.RevokedAt)
}
