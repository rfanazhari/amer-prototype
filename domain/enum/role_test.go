package enum_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/stretchr/testify/require"
)

// roleCase defines a fixed role token and whether it is a permitted administrative role.
type roleCase struct {
	name      string
	role      enum.Role
	wantValid bool
}

// TestRoleAcceptsOnlyFixedAdministrativeRoles rejects unknown, combined and case-variant roles.
func TestRoleAcceptsOnlyFixedAdministrativeRoles(t *testing.T) {
	tests := []roleCase{
		{"admin", enum.Admin, true},
		{"hr", enum.HR, true},
		{"empty", "", false},
		{"candidate", enum.Role("CANDIDATE"), false},
		{"combined", enum.Role("ADMIN,HR"), false},
		{"lowercase admin", enum.Role("admin"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantValid, tt.role.Valid())
			if tt.wantValid {
				require.Equal(t, string(tt.role), tt.role.String())
			}
		})
	}
}
