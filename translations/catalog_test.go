package translations_test

import (
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/translations"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"unicode"
)

// TestEveryErrorCodeHasBothTranslations checks locale coverage, lowercase messages and the default language for every public code.
func TestEveryErrorCodeHasBothTranslations(t *testing.T) {
	for _, code := range []string{
		errs.InvalidInput, errs.NotFound, errs.MethodNotAllowed, errs.InvalidTransition, errs.EventUnavailable, errs.InternalError, errs.ShuttingDown, errs.SessionUnavailable,
		errs.IAUnauthenticated, errs.IAForbidden, errs.IAAccountConflict, errs.IAInvitationUnavailable, errs.IARateLimited, errs.IADependencyUnavailable,
		errs.ACPUnauthenticated, errs.ACPForbidden, errs.ACPIdempotencyConflict, errs.ACPDraftConflict, errs.ACPValidationConflict, errs.ACPPublicationBlocked, errs.ACPDependencyUnavailable, errs.ACPPublicationUncertain,
		errs.EXLNotFound, errs.EXLAssignmentUnavailable, errs.EXLIdempotencyConflict, errs.EXLInvalidTransition, errs.EXLStaleVersion,
		errs.EXLStaleAttempt, errs.EXLProgressRequired, errs.EXLTerminalConflict, errs.EXLDependencyUnavailable, errs.EXLUncertainCommit,
	} {
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

// localeMessageCase is one localized catalog string for a public error code.
type localeMessageCase struct {
	name, locale, code, message string
}

// TestSessionUnavailableMessages locks the English and Indonesian voice-setup failure text.
func TestSessionUnavailableMessages(t *testing.T) {
	tests := []localeMessageCase{
		{name: "english", locale: "en", code: errs.SessionUnavailable, message: "voice session unavailable"},
		{name: "indonesian", locale: "id", code: errs.SessionUnavailable, message: "sesi suara tidak tersedia"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.message, translations.Message(tt.locale, tt.code))
		})
	}
}

// TestFindingCodesHaveBothTranslations covers T5 validation finding codes with safe EN/ID messages.
func TestFindingCodesHaveBothTranslations(t *testing.T) {
	for _, code := range catalog.FindingCodes() {
		t.Run(code, func(t *testing.T) {
			en := translations.Message("en", code)
			id := translations.Message("id", code)
			require.NotEmpty(t, en)
			require.NotEmpty(t, id)
			require.NotEqual(t, en, id)
			require.NotEqual(t, "internal error", en)
			require.False(t, unicode.IsUpper([]rune(en)[0]))
			require.False(t, unicode.IsUpper([]rune(id)[0]))
			require.Equal(t, en, translations.Message("", code))
			require.NotContains(t, strings.ToLower(en), "salary")
			require.NotContains(t, strings.ToLower(id), "salary")
		})
	}
}
