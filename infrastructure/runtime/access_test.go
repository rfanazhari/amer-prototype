package runtime_test

import (
	"testing"

	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

const (
	validCSRFSecret = "abcdefghijklmnopqrstuvwxyz012345"
	validPassword   = "correct-horse"
)

// accessSettingsCase names one bootstrap/origin/CSRF configuration outcome.
type accessSettingsCase struct {
	name    string
	cfg     appruntime.Config
	wantErr bool
	secret  string
}

// validAccessConfig returns complete administrative startup settings without real network resources.
func validAccessConfig() appruntime.Config {
	return appruntime.Config{
		AccessBootstrapAdminName:     "Sinta Wijaya",
		AccessBootstrapAdminEmail:    "sinta@example.com",
		AccessBootstrapAdminPassword: validPassword,
		AccessHTTPOrigin:             "http://localhost:8080",
		AccessCSRFSecret:             validCSRFSecret,
	}
}

// TestValidateAccessSettingsRejectsSecretsAndInvalidOrigins fails closed without echoing secrets.
func TestValidateAccessSettingsRejectsSecretsAndInvalidOrigins(t *testing.T) {
	valid := validAccessConfig()
	missingOrigin := valid
	missingOrigin.AccessHTTPOrigin = ""
	pathOrigin := valid
	pathOrigin.AccessHTTPOrigin = "http://localhost:8080/app"
	shortSecret := valid
	shortSecret.AccessCSRFSecret = "too-short-secret"
	tests := []accessSettingsCase{
		{name: "missing origin", cfg: missingOrigin, wantErr: true, secret: validPassword},
		{name: "origin with path", cfg: pathOrigin, wantErr: true, secret: validPassword},
		{name: "short csrf secret", cfg: shortSecret, wantErr: true, secret: "too-short-secret"},
		{name: "valid settings", cfg: valid, secret: validPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := appruntime.ValidateAccessSettings(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				require.NotContains(t, err.Error(), tt.secret)
				require.NotContains(t, err.Error(), validPassword)
				require.NotContains(t, err.Error(), validCSRFSecret)
				return
			}
			require.NoError(t, err)
		})
	}
}
