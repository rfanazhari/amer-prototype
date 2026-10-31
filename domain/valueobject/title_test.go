package valueobject_test

import (
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

// titleCase defines a normalized title or invalid Unicode/length input.
type titleCase struct {
	name, input, want string
	wantErr           bool
}

// TestTitle checks Unicode normalization, code-point limits, invalid encodings and value equality.
func TestTitle(t *testing.T) {
	tests := []titleCase{{"trim unicode whitespace", "\u2003 hello \t", "hello", false}, {"empty", "", "", true}, {"whitespace", " \u2003", "", true}, {"hundred code points", strings.Repeat("界", 100), strings.Repeat("界", 100), false}, {"too long", strings.Repeat("界", 101), "", true}, {"invalid utf8", string([]byte{0xff}), "", true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.NewTitle(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, valueobject.ErrInvalidTitle)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.String())
			same, err := valueobject.NewTitle(tt.want)
			require.NoError(t, err)
			require.Equal(t, same, got)
		})
	}
}
