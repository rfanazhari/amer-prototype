package translations_test

import (
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/translations"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"unicode"
)

// TestEveryErrorCodeHasBothTranslations checks locale coverage, lowercase messages and the default language for every public code.
func TestEveryErrorCodeHasBothTranslations(t *testing.T) {
	for _, code := range []string{errs.InvalidInput, errs.NotFound, errs.MethodNotAllowed, errs.InvalidTransition, errs.EventUnavailable, errs.InternalError, errs.ShuttingDown} {
		t.Run(code, func(t *testing.T) {
			en := translations.Message("en", code)
			id := translations.Message("id", code)
			require.NotEmpty(t, en)
			require.NotEmpty(t, id)
			require.NotEqual(t, en, id)
			require.False(t, unicode.IsUpper([]rune(en)[0]))
			require.False(t, unicode.IsUpper([]rune(id)[0]))
			require.Equal(t, en, translations.Message("", code))
			require.NotContains(t, strings.ToLower(en), "password")
		})
	}
}
