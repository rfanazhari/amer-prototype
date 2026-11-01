package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// classifiedCredentialCase names a credential failure that must stay free of secrets.
type classifiedCredentialCase struct {
	name    string
	err     error
	message string
	secret  string
	wantErr bool
}

// TestAccessCredentialFailuresHideSecrets keeps proof and CSRF failures free of passwords, hashes and tokens.
func TestAccessCredentialFailuresHideSecrets(t *testing.T) {
	tests := []classifiedCredentialCase{
		{name: "password mismatch", err: service.ErrAuthenticationFailed, message: "administrative authentication failed", secret: "correct-horse", wantErr: true},
		{name: "malformed hash", err: service.ErrMalformedCredential, message: "stored credential is malformed", secret: testdata.PasswordHash, wantErr: true},
		{name: "csrf mismatch", err: service.ErrCSRFMismatch, message: "csrf token does not match the current session", secret: "signed-csrf-material", wantErr: true},
		{name: "session unbound", err: service.ErrSessionUnbound, message: "administrative session is not bound to a current account", secret: testdata.TokenHash, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantErr, tt.err != nil)
			assert.Equal(t, tt.message, tt.err.Error())
			assert.NotContains(t, tt.err.Error(), tt.secret)
			assert.NotContains(t, tt.err.Error(), testdata.PasswordHash)
			assert.False(t, errors.Is(tt.err, errors.New(tt.message)))
		})
	}
}

// hasherVerifyCase names a generated PasswordHasher mismatch outcome.
type hasherVerifyCase struct {
	name    string
	verify  error
	wantErr bool
}

// TestPasswordHasherPortClassifiesMismatchUsesGeneratedMock proves verify failures stay classified at the port.
func TestPasswordHasherPortClassifiesMismatchUsesGeneratedMock(t *testing.T) {
	tests := []hasherVerifyCase{
		{name: "matching password", verify: nil, wantErr: false},
		{name: "mismatch", verify: service.ErrAuthenticationFailed, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := mocks.NewPasswordHasher(t)
			hasher.On("Verify", mock.Anything, testdata.PasswordHash, "correct-horse").Return(tt.verify).Once()
			err := hasher.Verify(context.Background(), testdata.PasswordHash, "correct-horse")
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				require.ErrorIs(t, err, service.ErrAuthenticationFailed)
			}
		})
	}
}

// currentSessionCase names trusted current-session binding outcomes.
type currentSessionCase struct {
	name    string
	resolve error
	wantErr bool
}

// TestCurrentAdministrativeSessionRequiresAccountBinding rejects unbound sessions through the generated mock.
func TestCurrentAdministrativeSessionRequiresAccountBinding(t *testing.T) {
	session := testdata.AdministrativeSessionRecord(t)
	account := testdata.AdministrativeAccountRecord(t)
	tests := []currentSessionCase{
		{name: "current bound session", resolve: nil, wantErr: false},
		{name: "unbound session", resolve: service.ErrSessionUnbound, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := mocks.NewCurrentAdministrativeSession(t)
			resolver.On("Resolve", mock.Anything).Return(session, account, tt.resolve).Once()
			gotSession, gotAccount, err := resolver.Resolve(context.Background())
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				require.ErrorIs(t, err, service.ErrSessionUnbound)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, session, gotSession)
			assert.Equal(t, account, gotAccount)
		})
	}
}

// TestAgentConfigurationAuthorityShapeIsUnchanged preserves Principal and Authorize without a caller role field.
func TestAgentConfigurationAuthorityShapeIsUnchanged(t *testing.T) {
	authority := mocks.NewAgentConfigurationAuthority(t)
	principal := service.AdminPrincipal{SubjectID: testdata.ID}
	authority.On("Principal", mock.Anything).Return(principal, nil).Once()
	authority.On("Authorize", mock.Anything, principal, "create", "").Return(nil).Once()
	var port service.AgentConfigurationAuthority = authority
	got, err := port.Principal(context.Background())
	require.NoError(t, err)
	assert.Equal(t, testdata.ID, got.SubjectID)
	require.NoError(t, port.Authorize(context.Background(), got, "create", ""))
}
