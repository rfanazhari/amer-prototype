package accessusecase

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// restoreAccount rebuilds the administrative account aggregate from a detached record.
func restoreAccount(record accessport.AccountRecord) (*access.AdministrativeAccount, error) {
	return access.RestoreAdministrativeAccount(access.AccountSnapshot{
		ID:        record.ID,
		Role:      record.Role,
		Status:    record.Status,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	})
}

// restoreSession rebuilds the administrative session aggregate from a detached record.
func restoreSession(record accessport.SessionRecord) (*access.AdministrativeSession, error) {
	return access.RestoreAdministrativeSession(access.SessionSnapshot{
		ID:        record.ID,
		AccountID: record.AccountID,
		CreatedAt: record.CreatedAt,
		ExpiresAt: record.ExpiresAt,
		RevokedAt: record.RevokedAt,
	})
}

// authorizeCurrent rejects expired, revoked, disabled or mismatched session/account pairs.
func authorizeCurrent(session accessport.SessionRecord, account accessport.AccountRecord, now time.Time) error {
	restoredAccount, err := restoreAccount(account)
	if err != nil {
		return err
	}
	restoredSession, err := restoreSession(session)
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
