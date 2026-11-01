package agentconfig_test

import (
	"testing"

	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// documentFindingCase names a required-field subset and its publication findings.
type documentFindingCase struct {
	name     string
	fields   []string
	eligible bool
	findings []aggregate.Finding
}

// TestAssessDocumentCoversSubsetWarningAndEmptyDrafts retains informational currency warnings.
func TestAssessDocumentCoversSubsetWarningAndEmptyDrafts(t *testing.T) {
	tests := []documentFindingCase{
		{name: "valid subset", fields: []string{"invoice_number", "total", "currency"}, eligible: true},
		{name: "empty draft", fields: []string{}, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeRequiredFieldsEmpty, "/requiredFields", "at least one required invoice field must be selected"),
		}},
		{name: "total without currency", fields: []string{"total"}, eligible: true, findings: []aggregate.Finding{
			finding(enum.FindingWarning, catalog.CodeCurrencyNotRequired, "/requiredFields", "total is required without currency"),
		}},
		{name: "total with currency", fields: []string{"total", "currency"}, eligible: true},
		{name: "currency only", fields: []string{"currency"}, eligible: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := valueobject.NewDocumentConfiguration(tt.fields)
			require.NoError(t, err)
			findings := catalog.AssessDocument(cfg)
			assert.Equal(t, tt.findings, findings)
			assert.Equal(t, tt.eligible, catalog.Eligible(findings))
		})
	}
}
