package agentconfigport_test

import (
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/stretchr/testify/assert"
)

// TestCommitUncertainError preserves internal diagnostics without exposing them in its message.
func TestCommitUncertainError(t *testing.T) {
	// testCase describes an uncertain commit cause.
	type testCase struct {
		name    string
		cause   error
		wantErr bool
	}
	tests := []testCase{
		{name: "storage failure", cause: errors.New("private database detail"), wantErr: true},
		{name: "unknown cause", cause: nil, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &agentconfigport.CommitUncertainError{Cause: tt.cause}
			assert.Equal(t, "publication commit outcome uncertain", err.Error())
			assert.Equal(t, tt.wantErr, errors.Unwrap(err) != nil)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.cause)
			}
		})
	}
}
