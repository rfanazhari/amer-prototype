package enum_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/stretchr/testify/require"
)

// findingSeverityCase names a severity token and whether it may appear on a finding.
type findingSeverityCase struct {
	name      string
	severity  enum.FindingSeverity
	wantValid bool
}

// TestFindingSeverityAcceptsOnlyErrorAndWarning rejects unknown severity tokens.
func TestFindingSeverityAcceptsOnlyErrorAndWarning(t *testing.T) {
	tests := []findingSeverityCase{
		{name: "error", severity: enum.FindingError, wantValid: true},
		{name: "warning", severity: enum.FindingWarning, wantValid: true},
		{name: "empty", severity: "", wantValid: false},
		{name: "info", severity: enum.FindingSeverity("info"), wantValid: false},
		{name: "uppercase error", severity: enum.FindingSeverity("ERROR"), wantValid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantValid, tt.severity.Valid())
			if tt.wantValid {
				require.Equal(t, string(tt.severity), tt.severity.String())
			}
		})
	}
}
