package enum

// InvoiceField is a supported Document Analyzer required-extraction choice.
type InvoiceField string

const (
	InvoiceNumber   InvoiceField = "invoice_number"
	InvoiceVendor   InvoiceField = "vendor"
	InvoiceDate     InvoiceField = "date"
	InvoiceTotal    InvoiceField = "total"
	InvoiceCurrency InvoiceField = "currency"
)

// Valid reports whether the field is one of the five catalog invoice fields.
func (f InvoiceField) Valid() bool {
	return f == InvoiceNumber || f == InvoiceVendor || f == InvoiceDate || f == InvoiceTotal || f == InvoiceCurrency
}

// String returns the stable storage and API representation of the field.
func (f InvoiceField) String() string { return string(f) }

// InvoiceFieldCatalog returns the supported fields in catalog order.
func InvoiceFieldCatalog() []InvoiceField {
	return []InvoiceField{InvoiceNumber, InvoiceVendor, InvoiceDate, InvoiceTotal, InvoiceCurrency}
}
