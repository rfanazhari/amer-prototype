package enum_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/stretchr/testify/require"
)

// accountStatusCase defines a fixed account status token and whether it is permitted.
type accountStatusCase struct {
	name      string
	status    enum.AccountStatus
	wantValid bool
}

// TestAccountStatusAcceptsOnlyFixedStates rejects unknown and case-variant account statuses.
func TestAccountStatusAcceptsOnlyFixedStates(t *testing.T) {
	tests := []accountStatusCase{
		{"active", enum.Active, true},
		{"disabled", enum.Disabled, true},
		{"empty", "", false},
		{"unknown", enum.AccountStatus("UNKNOWN"), false},
		{"lowercase active", enum.AccountStatus("active"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantValid, tt.status.Valid())
			if tt.wantValid {
				require.Equal(t, string(tt.status), tt.status.String())
			}
		})
	}
}
