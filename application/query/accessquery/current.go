package accessquery

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// CurrentSessionOperation returns the current administrative session after trusted binding.
type CurrentSessionOperation interface {
	// Execute returns detached account, session and CSRF material for a current bound session.
	Execute(ctx context.Context) (*dto.AdministrativeCurrentSession, error)
}

// CurrentSession returns the current administrative session after trusted binding.
type CurrentSession struct {
	Current service.CurrentAdministrativeSession
	CSRF    service.CSRFTokens
	Clock   service.Clock
}

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

// Execute returns detached account, session and CSRF material for a current bound session.
func (q CurrentSession) Execute(ctx context.Context) (*dto.AdministrativeCurrentSession, error) {
	session, account, err := q.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	now := q.Clock.Now().UTC().Truncate(time.Millisecond)
	if err := authorizeCurrent(session, account, now); err != nil {
		return nil, err
	}
	token, err := q.CSRF.Issue(service.CredentialAdministrative, session.ID)
	if err != nil {
		return nil, err
	}
	return &dto.AdministrativeCurrentSession{
		Account:   dto.FromAdministrativeAccount(account),
		Session:   dto.FromAdministrativeSession(session),
		CSRFToken: token,
	}, nil
}
