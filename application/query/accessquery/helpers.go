package accessquery

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// trustedQueryNow canonicalizes injected clock samples to UTC milliseconds.
func trustedQueryNow(clock service.Clock) time.Time {
	return clock.Now().UTC().Truncate(time.Millisecond)
}

// authorizeAdministrative rejects expired, revoked, disabled or mismatched administrative pairs.
func authorizeAdministrative(session accessport.SessionRecord, account accessport.AccountRecord, now time.Time) error {
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
