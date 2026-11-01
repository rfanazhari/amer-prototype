package service

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
)

// CredentialKind names an independent credential generation and lookup namespace.
type CredentialKind string

const (
	// CredentialAdministrative binds administrative session cookies and CSRF values.
	CredentialAdministrative CredentialKind = "administrative"
	// CredentialCandidate binds candidate session cookies and CSRF values.
	CredentialCandidate CredentialKind = "candidate"
	// CredentialInvitation binds assignment invitation bearer secrets.
	CredentialInvitation CredentialKind = "invitation"
)

var (
	// ErrAuthenticationFailed is the generic administrative proof failure.
	ErrAuthenticationFailed = errors.New("pending")
	// ErrMalformedCredential reports an unusable stored hash without allocating unbounded work.
	ErrMalformedCredential = errors.New("pending")
	// ErrCSRFMismatch reports a session-bound CSRF value that does not match.
	ErrCSRFMismatch = errors.New("pending")
	// ErrSessionUnbound reports a missing, expired, revoked or account-mismatched session.
	ErrSessionUnbound = errors.New("pending")
)

// PasswordHasher hashes and verifies administrative passwords outside persistence transactions.
type PasswordHasher interface {
	// Hash returns an encoded Argon2id digest for a plaintext password.
	Hash(ctx context.Context, password string) (string, error)
	// Verify compares a stored encoded digest with a plaintext password.
	Verify(ctx context.Context, encoded, password string) error
}

// TokenSecrets issues independent high-entropy secrets and digests per credential kind.
type TokenSecrets interface {
	// Issue returns a fresh plaintext secret and its persisted digest for kind.
	Issue(ctx context.Context, kind CredentialKind) (plaintext, digest string, err error)
	// Digest maps a presented secret into the lookup digest for kind.
	Digest(kind CredentialKind, plaintext string) (string, error)
}

// CSRFTokens issues and checks HMAC values bound to credential kind and session identity.
type CSRFTokens interface {
	// Issue returns CSRF material for the current session and credential kind.
	Issue(kind CredentialKind, sessionID string) (string, error)
	// Verify accepts CSRF material only for the matching kind and session identity.
	Verify(kind CredentialKind, sessionID, token string) error
}

// CurrentAdministrativeSession resolves the trusted administrative session and bound account.
type CurrentAdministrativeSession interface {
	// Resolve returns the current session and account or ErrSessionUnbound.
	Resolve(ctx context.Context) (accessport.SessionRecord, accessport.AccountRecord, error)
}
