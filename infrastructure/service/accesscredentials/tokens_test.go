package accesscredentials_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tokenIssueCase names independent credential-kind issuance.
type tokenIssueCase struct {
	name    string
	kind    service.CredentialKind
	wantErr bool
}

// TestTokensIssueIndependentSecretsAndDigests keeps invitation and session namespaces distinct.
func TestTokensIssueIndependentSecretsAndDigests(t *testing.T) {
	tests := []tokenIssueCase{
		{name: "administrative session", kind: service.CredentialAdministrative, wantErr: false},
		{name: "invitation", kind: service.CredentialInvitation, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstPlain, firstDigest, err := accesscredentials.Tokens{}.Issue(context.Background(), tt.kind)
			require.NoError(t, err)
			secondPlain, secondDigest, err := accesscredentials.Tokens{}.Issue(context.Background(), tt.kind)
			require.NoError(t, err)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.NotEmpty(t, firstPlain)
			assert.NotEqual(t, firstPlain, secondPlain)
			assert.Len(t, firstDigest, 64)
			assert.NotEqual(t, firstDigest, secondDigest)
			assert.NotContains(t, firstDigest, firstPlain)
			mapped, err := accesscredentials.Tokens{}.Digest(tt.kind, firstPlain)
			require.NoError(t, err)
			assert.Equal(t, firstDigest, mapped)
			otherKind := service.CredentialCandidate
			if tt.kind == service.CredentialCandidate {
				otherKind = service.CredentialAdministrative
			}
			crossed, err := accesscredentials.Tokens{}.Digest(otherKind, firstPlain)
			require.NoError(t, err)
			assert.NotEqual(t, firstDigest, crossed)
		})
	}
}

// malformedTokenCase names a presented secret that must not yield a digest.
type malformedTokenCase struct {
	name    string
	token   string
	wantErr bool
}

// TestTokensDigestRejectsMalformedSecrets fails closed without echoing the presented value.
func TestTokensDigestRejectsMalformedSecrets(t *testing.T) {
	tests := []malformedTokenCase{
		{name: "empty", token: "", wantErr: true},
		{name: "not base64url", token: "%%%not-a-token%%%", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digest, err := accesscredentials.Tokens{}.Digest(service.CredentialAdministrative, tt.token)
			assert.Equal(t, tt.wantErr, err != nil)
			require.ErrorIs(t, err, service.ErrMalformedCredential)
			assert.Empty(t, digest)
			if tt.token != "" {
				assert.NotContains(t, err.Error(), tt.token)
			}
		})
	}
}
