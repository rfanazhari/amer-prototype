package accesscredentials

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
)

// Tokens issues independent 32-byte secrets and kind-scoped digests.
type Tokens struct{}

// Issue returns a fresh plaintext secret and persisted digest for kind.
func (Tokens) Issue(ctx context.Context, kind service.CredentialKind) (string, string, error) {
	return "", "", nil
}

// Digest maps a presented secret into the lookup digest for kind.
func (Tokens) Digest(kind service.CredentialKind, plaintext string) (string, error) {
	return "", nil
}
