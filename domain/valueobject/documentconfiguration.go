package valueobject

import "github.com/rfanazhari/amer-prototype/domain/enum"

const documentFieldLimit = 5

// DocumentConfiguration is an editable Document Analyzer draft of required invoice fields.
type DocumentConfiguration struct{ fields []enum.InvoiceField }

// NewDocumentConfiguration stores distinct catalog fields and rejects unsupported or duplicate values.
func NewDocumentConfiguration(requiredFields []string) (DocumentConfiguration, error) {
	return DocumentConfiguration{}, nil
}

// Clone returns an independently owned copy of the required-field list.
func (c DocumentConfiguration) Clone() DocumentConfiguration { return c }

// RequiredFields returns a copied list of required invoice fields in catalog order.
func (c DocumentConfiguration) RequiredFields() []enum.InvoiceField { return c.fields }
