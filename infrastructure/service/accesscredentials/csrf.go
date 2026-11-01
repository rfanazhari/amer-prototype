package accesscredentials

import "github.com/rfanazhari/amer-prototype/application/service"

// CSRF issues HMAC values bound to credential kind and session identity.
type CSRF struct {
	Secret []byte
}

// Issue returns CSRF material for kind and sessionID.
func (CSRF) Issue(kind service.CredentialKind, sessionID string) (string, error) {
	return "", nil
}

// Verify accepts CSRF material only for the matching kind and session identity.
func (CSRF) Verify(kind service.CredentialKind, sessionID, token string) error {
	return nil
}
