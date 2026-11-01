package accesscredentials_test

import (
	"bytes"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// csrfCase names a CSRF bind or reject outcome.
type csrfCase struct {
	name      string
	issueKind service.CredentialKind
	checkKind service.CredentialKind
	session   string
	checkSess string
	wantErr   bool
}

// TestCSRFBindsKindAndSession rejects swapped credential kinds and other session identities.
func TestCSRFBindsKindAndSession(t *testing.T) {
	secret := bytes.Repeat([]byte("s"), 32)
	csrf := accesscredentials.CSRF{Secret: secret}
	tests := []csrfCase{
		{name: "matching administrative session", issueKind: service.CredentialAdministrative, checkKind: service.CredentialAdministrative, session: testdata.SessionID, checkSess: testdata.SessionID, wantErr: false},
		{name: "candidate kind rejected", issueKind: service.CredentialAdministrative, checkKind: service.CredentialCandidate, session: testdata.SessionID, checkSess: testdata.SessionID, wantErr: true},
		{name: "other session rejected", issueKind: service.CredentialAdministrative, checkKind: service.CredentialAdministrative, session: testdata.SessionID, checkSess: testdata.ID, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := csrf.Issue(tt.issueKind, tt.session)
			require.NoError(t, err)
			assert.NotEmpty(t, token)
			err = csrf.Verify(tt.checkKind, tt.checkSess, token)
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				require.ErrorIs(t, err, service.ErrCSRFMismatch)
				assert.NotContains(t, err.Error(), token)
				assert.NotContains(t, err.Error(), string(secret))
				assert.NotContains(t, err.Error(), tt.session)
				return
			}
			require.NoError(t, err)
		})
	}
}
