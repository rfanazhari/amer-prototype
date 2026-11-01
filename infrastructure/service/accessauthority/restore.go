package accessauthority

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// authorizeCurrent rejects expired, revoked, disabled or mismatched session/account pairs.
func authorizeCurrent(session accessport.SessionRecord, account accessport.AccountRecord, now time.Time) error {
	restoredAccount, err := access.RestoreAdministrativeAccount(access.AccountSnapshot{
		ID:        account.ID,
		Role:      account.Role,
		Status:    account.Status,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	})
	if err != nil {
		return err
	}
	restoredSession, err := access.RestoreAdministrativeSession(access.SessionSnapshot{
		ID:        session.ID,
		AccountID: session.AccountID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
		RevokedAt: session.RevokedAt,
	})
	if err != nil {
		return err
	}
	if !restoredSession.Authorizes(restoredAccount, now) {
		return service.ErrSessionUnbound
	}
	return nil
}

// trustedNow canonicalizes injected clock samples to UTC milliseconds.
func trustedNow(clock service.Clock) time.Time {
	return clock.Now().UTC().Truncate(time.Millisecond)
}
