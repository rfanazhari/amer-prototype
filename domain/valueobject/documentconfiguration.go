package valueobject

import "github.com/rfanazhari/amer-prototype/domain/enum"

const documentFieldLimit = 5

// DocumentConfiguration is an editable Document Analyzer draft of required invoice fields.
type DocumentConfiguration struct{ fields []enum.InvoiceField }

// NewDocumentConfiguration stores distinct catalog fields and rejects unsupported or duplicate values.
func NewDocumentConfiguration(requiredFields []string) (DocumentConfiguration, error) {
	if len(requiredFields) > documentFieldLimit {
		return DocumentConfiguration{}, ErrInvalidConfiguration
	}
	seen := make(map[enum.InvoiceField]struct{}, len(requiredFields))
	selected := make(map[enum.InvoiceField]struct{}, len(requiredFields))
	for _, value := range requiredFields {
		field := enum.InvoiceField(value)
		if !field.Valid() {
			return DocumentConfiguration{}, ErrInvalidConfiguration
		}
		if _, exists := seen[field]; exists {
			return DocumentConfiguration{}, ErrInvalidConfiguration
		}
		seen[field] = struct{}{}
		selected[field] = struct{}{}
	}
	out := []enum.InvoiceField{}
	for _, field := range enum.InvoiceFieldCatalog() {
		if _, ok := selected[field]; ok {
			out = append(out, field)
		}
	}
	return DocumentConfiguration{fields: out}, nil
}

// Clone returns an independently owned copy of the required-field list.
func (c DocumentConfiguration) Clone() DocumentConfiguration {
	return DocumentConfiguration{fields: copyInvoiceFields(c.fields)}
}

// RequiredFields returns a copied list of required invoice fields in catalog order.
func (c DocumentConfiguration) RequiredFields() []enum.InvoiceField {
	return copyInvoiceFields(c.fields)
}

// copyInvoiceFields returns an independently owned field list, never a nil empty slice.
func copyInvoiceFields(fields []enum.InvoiceField) []enum.InvoiceField {
	out := make([]enum.InvoiceField, len(fields))
	copy(out, fields)
	return out
}
