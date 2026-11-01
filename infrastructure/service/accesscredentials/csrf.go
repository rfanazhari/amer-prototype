package accesscredentials

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"github.com/rfanazhari/amer-prototype/application/service"
)

const csrfSecretMinLen = 32

var _ service.CSRFTokens = CSRF{}

// signedCSRF returns HMAC-SHA256 material bound to kind and session identity.
func signedCSRF(secret []byte, kind service.CredentialKind, sessionID string) (string, error) {
	if len(secret) < csrfSecretMinLen || kind == "" || sessionID == "" {
		return "", service.ErrCSRFMismatch
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(kind))
	_, _ = mac.Write([]byte{0x1f})
	_, _ = mac.Write([]byte(sessionID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// CSRF issues HMAC values bound to credential kind and session identity.
type CSRF struct {
	Secret []byte
}

// Issue returns CSRF material for kind and sessionID.
func (c CSRF) Issue(kind service.CredentialKind, sessionID string) (string, error) {
	return signedCSRF(c.Secret, kind, sessionID)
}

// Verify accepts CSRF material only for the matching kind and session identity.
func (c CSRF) Verify(kind service.CredentialKind, sessionID, token string) error {
	expected, err := signedCSRF(c.Secret, kind, sessionID)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected), []byte(token)) {
		return service.ErrCSRFMismatch
	}
	return nil
}
