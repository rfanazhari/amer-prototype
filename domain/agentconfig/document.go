package agentconfig

import (
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// AssessDocument records deterministic Document Analyzer findings without provider calls.
func AssessDocument(cfg valueobject.DocumentConfiguration) []aggregate.Finding {
	fields := cfg.RequiredFields()
	if len(fields) == 0 {
		findings := []aggregate.Finding{errorFinding(CodeRequiredFieldsEmpty, "/requiredFields", messageRequiredFieldsEmpty)}
		sortFindings(findings)
		return findings
	}
	hasTotal, hasCurrency := false, false
	for _, field := range fields {
		switch field {
		case enum.InvoiceTotal:
			hasTotal = true
		case enum.InvoiceCurrency:
			hasCurrency = true
		}
	}
	if !hasTotal || hasCurrency {
		return nil
	}
	findings := []aggregate.Finding{warningFinding(CodeCurrencyNotRequired, "/requiredFields", messageCurrencyNotRequired)}
	sortFindings(findings)
	return findings
}
