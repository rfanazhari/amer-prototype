package valueobject_test

import (
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stableRefCase names a reference identity and version that must be accepted or rejected.
type stableRefCase struct {
	name, id, version string
	wantErr           bool
}

// TestNewStableRefAcceptsExactUUIDVersions rejects malformed identities and non-positive versions.
func TestNewStableRefAcceptsExactUUIDVersions(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	tests := []stableRefCase{
		{name: "canonical", id: id, version: "1", wantErr: false},
		{name: "uppercase uuid", id: "11111111-1111-4111-8111-11111111111A", version: "1", wantErr: false},
		{name: "ten digit version", id: id, version: "1234567890", wantErr: false},
		{name: "empty id", id: "", version: "1", wantErr: true},
		{name: "short uuid", id: "11111111-1111-4111-8111-11111111111", version: "1", wantErr: true},
		{name: "missing hyphen", id: "11111111111141118111111111111111", version: "1", wantErr: true},
		{name: "non hex", id: "11111111-1111-4111-8111-11111111111g", version: "1", wantErr: true},
		{name: "empty version", id: id, version: "", wantErr: true},
		{name: "zero version", id: id, version: "0", wantErr: true},
		{name: "leading zero", id: id, version: "01", wantErr: true},
		{name: "eleven digits", id: id, version: "12345678901", wantErr: true},
		{name: "plus prefix", id: id, version: "+1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.NewStableRef(tt.id, tt.version)
			if tt.wantErr {
				require.ErrorIs(t, err, valueobject.ErrInvalidConfiguration)
				assert.Equal(t, valueobject.StableRef{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, strings.ToLower(tt.id), got.ID())
			assert.Equal(t, tt.version, got.Version())
		})
	}
}
