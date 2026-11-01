package accessusecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// Login establishes an administrative session after credential proof and commit.
type Login struct {
	Accounts accessport.Accounts
	Issuance accessport.SessionIssuance
	Hasher   service.PasswordHasher
	Tokens   service.TokenSecrets
	Clock    service.Clock
}

// failAuthentication runs equivalent hasher work for unknown accounts then returns a generic failure.
func failAuthentication(ctx context.Context, hasher service.PasswordHasher, encoded, password string) error {
	if err := hasher.Verify(ctx, encoded, password); err != nil && !errors.Is(err, service.ErrAuthenticationFailed) && !errors.Is(err, service.ErrMalformedCredential) {
		return err
	}
	return service.ErrAuthenticationFailed
}

// Execute returns detached account and session metadata only after session commit.
func (u Login) Execute(ctx context.Context, email, password string) (*dto.AdministrativeLogin, error) {
	if err := service.AcceptLoginPassword(password); err != nil {
		return nil, err
	}
	canonicalEmail, err := service.CanonicalAdministrativeEmail(email)
	if err != nil {
		return nil, err
	}
	account, err := u.Accounts.FindByEmail(ctx, canonicalEmail)
	if errors.Is(err, accessport.ErrNotFound) {
		return nil, failAuthentication(ctx, u.Hasher, service.DummyEncodedPassword, password)
	}
	if err != nil {
		return nil, err
	}
	if err := u.Hasher.Verify(ctx, account.PasswordHash, password); err != nil {
		if errors.Is(err, service.ErrAuthenticationFailed) || errors.Is(err, service.ErrMalformedCredential) {
			return nil, service.ErrAuthenticationFailed
		}
		return nil, err
	}
	if account.Status != enum.Active {
		return nil, service.ErrAuthenticationFailed
	}
	now := trustedNow(u.Clock)
	_, digest, err := u.Tokens.Issue(ctx, service.CredentialAdministrative)
	if err != nil {
		return nil, err
	}
	session, err := access.NewAdministrativeSession(uuid.NewString(), account.ID, now)
	if err != nil {
		return nil, err
	}
	snapshot := session.Snapshot()
	record := accessport.SessionRecord{
		ID:        snapshot.ID,
		AccountID: snapshot.AccountID,
		TokenHash: digest,
		CreatedAt: snapshot.CreatedAt,
		ExpiresAt: snapshot.ExpiresAt,
		RevokedAt: snapshot.RevokedAt,
	}
	var locked accessport.AccountRecord
	err = u.Issuance.WithinLogin(ctx, account.ID, func(tx accessport.LoginTx) error {
		var err error
		locked, err = tx.Account(ctx)
		if err != nil {
			return err
		}
		if locked.PasswordHash != account.PasswordHash || locked.Status != enum.Active {
			return service.ErrAuthenticationFailed
		}
		return tx.InsertSession(ctx, record)
	})
	if err != nil {
		return nil, err
	}
	return &dto.AdministrativeLogin{
		Account: dto.FromAdministrativeAccount(locked),
		Session: dto.FromAdministrativeSession(record),
	}, nil
}
