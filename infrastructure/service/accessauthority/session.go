package accessauthority

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Sessions resolves the current administrative session from a trusted context credential.
type Sessions struct {
	Store  accessport.Sessions
	Tokens service.TokenSecrets
	Clock  service.Clock
}

var _ service.CurrentAdministrativeSession = Sessions{}

// Resolve binds a trusted administrative secret to the current eligible account or reports unbound access.
func (s Sessions) Resolve(ctx context.Context) (accessport.SessionRecord, accessport.AccountRecord, error) {
	plaintext, ok := service.AdministrativeCredentialFrom(ctx)
	if !ok {
		return accessport.SessionRecord{}, accessport.AccountRecord{}, service.ErrSessionUnbound
	}
	digest, err := s.Tokens.Digest(service.CredentialAdministrative, plaintext)
	if err != nil {
		return accessport.SessionRecord{}, accessport.AccountRecord{}, service.ErrSessionUnbound
	}
	session, account, err := s.Store.FindWithAccountByTokenHash(ctx, digest)
	if errors.Is(err, accessport.ErrNotFound) {
		return accessport.SessionRecord{}, accessport.AccountRecord{}, service.ErrSessionUnbound
	}
	if err != nil {
		return accessport.SessionRecord{}, accessport.AccountRecord{}, err
	}
	if err := authorizeCurrent(session, account, trustedNow(s.Clock)); err != nil {
		return accessport.SessionRecord{}, accessport.AccountRecord{}, err
	}
	return session, account, nil
}
