package enum_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/stretchr/testify/require"
)

// invoiceFieldCase names a field token and whether it is a supported invoice choice.
type invoiceFieldCase struct {
	name      string
	field     enum.InvoiceField
	wantValid bool
}

// TestInvoiceFieldAcceptsOnlyCatalogValues rejects unsupported extraction fields.
func TestInvoiceFieldAcceptsOnlyCatalogValues(t *testing.T) {
	tests := []invoiceFieldCase{
		{name: "invoice number", field: enum.InvoiceNumber, wantValid: true},
		{name: "vendor", field: enum.InvoiceVendor, wantValid: true},
		{name: "date", field: enum.InvoiceDate, wantValid: true},
		{name: "total", field: enum.InvoiceTotal, wantValid: true},
		{name: "currency", field: enum.InvoiceCurrency, wantValid: true},
		{name: "empty", field: "", wantValid: false},
		{name: "credit score", field: enum.InvoiceField("customer_credit_score"), wantValid: false},
		{name: "uppercase total", field: enum.InvoiceField("TOTAL"), wantValid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantValid, tt.field.Valid())
			if tt.wantValid {
				require.Equal(t, string(tt.field), tt.field.String())
			}
		})
	}
	require.Equal(t, []enum.InvoiceField{
		enum.InvoiceNumber, enum.InvoiceVendor, enum.InvoiceDate, enum.InvoiceTotal, enum.InvoiceCurrency,
	}, enum.InvoiceFieldCatalog())
}
