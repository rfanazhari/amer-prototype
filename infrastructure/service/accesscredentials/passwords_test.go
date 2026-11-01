package accesscredentials_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const argon2idBaselinePrefix = "$argon2id$v=19$m=19456,t=2,p=1$"

// passwordCase names a hash or verify outcome for the Argon2id adapter.
type passwordCase struct {
	name     string
	password string
	verify   string
	encoded  string
	wantErr  bool
	class    error
}

// TestPasswordsHashAndVerify covers match, mismatch, malformed parameters and secret-free errors.
func TestPasswordsHashAndVerify(t *testing.T) {
	tests := []passwordCase{
		{name: "matching password", password: "correct-horse", verify: "correct-horse", wantErr: false},
		{name: "mismatch", password: "correct-horse", verify: "other-horse-1", wantErr: true, class: service.ErrAuthenticationFailed},
		{name: "costly stored parameters", encoded: "$argon2id$v=19$m=99999999,t=99,p=16$c2FsdHNhbHQ$c3RvcmVkLWhhc2g", verify: "correct-horse", wantErr: true, class: service.ErrMalformedCredential},
		{name: "wrong algorithm", encoded: "$argon2i$v=19$m=19456,t=2,p=1$c2FsdHNhbHQ$c3RvcmVkLWhhc2g", verify: "correct-horse", wantErr: true, class: service.ErrMalformedCredential},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			encoded := tt.encoded
			if encoded == "" {
				var err error
				encoded, err = accesscredentials.Passwords{}.Hash(ctx, tt.password)
				require.NoError(t, err)
				assert.True(t, strings.HasPrefix(encoded, argon2idBaselinePrefix))
				assert.NotContains(t, encoded, tt.password)
			}
			started := time.Now()
			err := accesscredentials.Passwords{}.Verify(ctx, encoded, tt.verify)
			assert.Less(t, time.Since(started), 800*time.Millisecond)
			if tt.wantErr {
				require.ErrorIs(t, err, tt.class)
				if tt.password != "" {
					assert.NotContains(t, err.Error(), tt.password)
				}
				if tt.verify != "" {
					assert.NotContains(t, err.Error(), tt.verify)
				}
				assert.NotContains(t, err.Error(), encoded)
				return
			}
			require.NoError(t, err)
		})
	}
}

// cancelledHashCase names a cancelled hashing attempt.
type cancelledHashCase struct {
	name    string
	wantErr bool
}

// TestPasswordsHashRespectsCancellation returns the context error without a digest.
func TestPasswordsHashRespectsCancellation(t *testing.T) {
	tests := []cancelledHashCase{
		{name: "already cancelled", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			encoded, err := accesscredentials.Passwords{}.Hash(ctx, "correct-horse")
			assert.Equal(t, tt.wantErr, err != nil)
			require.ErrorIs(t, err, context.Canceled)
			assert.Empty(t, encoded)
			assert.Equal(t, 2, accesscredentials.HashConcurrency)
		})
	}
}
