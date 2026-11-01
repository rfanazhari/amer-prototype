package accesscredentials

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/rfanazhari/amer-prototype/application/service"
)

const tokenByteLen = 32

var _ service.TokenSecrets = Tokens{}

// digestRaw hashes kind and raw secret bytes into a 64-character lowercase hex digest.
func digestRaw(kind service.CredentialKind, raw []byte) string {
	sum := sha256.New()
	_, _ = sum.Write([]byte(kind))
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write(raw)
	return hex.EncodeToString(sum.Sum(nil))
}

// Tokens issues independent 32-byte secrets and kind-scoped digests.
type Tokens struct{}

// Issue returns a fresh plaintext secret and persisted digest for kind.
func (Tokens) Issue(ctx context.Context, kind service.CredentialKind) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	raw := make([]byte, tokenByteLen)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	plain := base64.RawURLEncoding.EncodeToString(raw)
	return plain, digestRaw(kind, raw), nil
}

// Digest maps a presented secret into the lookup digest for kind.
func (Tokens) Digest(kind service.CredentialKind, plaintext string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(plaintext)
	if err != nil || len(raw) != tokenByteLen {
		return "", service.ErrMalformedCredential
	}
	return digestRaw(kind, raw), nil
}
