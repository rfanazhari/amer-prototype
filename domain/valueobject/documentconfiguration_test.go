package valueobject_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// documentConstructionCase names a required-field replacement and whether it is storage-valid.
type documentConstructionCase struct {
	name    string
	fields  []string
	want    []enum.InvoiceField
	wantErr bool
}

// TestNewDocumentConfigurationAcceptsIncompleteSubsets rejects unsupported and duplicate fields.
func TestNewDocumentConfigurationAcceptsIncompleteSubsets(t *testing.T) {
	tests := []documentConstructionCase{
		{name: "valid subset", fields: []string{"invoice_number", "total", "currency"}, want: []enum.InvoiceField{enum.InvoiceNumber, enum.InvoiceTotal, enum.InvoiceCurrency}},
		{name: "empty draft", fields: []string{}, want: []enum.InvoiceField{}},
		{name: "nil draft", fields: nil, want: []enum.InvoiceField{}},
		{name: "catalog order", fields: []string{"currency", "date", "vendor"}, want: []enum.InvoiceField{enum.InvoiceVendor, enum.InvoiceDate, enum.InvoiceCurrency}},
		{name: "unsupported field", fields: []string{"customer_credit_score"}, wantErr: true},
		{name: "duplicate total", fields: []string{"total", "total"}, wantErr: true},
		{name: "too many fields", fields: []string{"invoice_number", "vendor", "date", "total", "currency", "invoice_number"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.NewDocumentConfiguration(tt.fields)
			if tt.wantErr {
				require.ErrorIs(t, err, valueobject.ErrInvalidConfiguration)
				assert.Equal(t, valueobject.DocumentConfiguration{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.RequiredFields())
		})
	}
}

// TestDocumentConfigurationCopyIsolation keeps required-field slices independently owned.
func TestDocumentConfigurationCopyIsolation(t *testing.T) {
	input := []string{"total", "invoice_number"}
	cfg, err := valueobject.NewDocumentConfiguration(input)
	require.NoError(t, err)
	input[0] = "mutated"
	assert.Equal(t, []enum.InvoiceField{enum.InvoiceNumber, enum.InvoiceTotal}, cfg.RequiredFields())
	require.NotEmpty(t, cfg.RequiredFields())
	fields := cfg.RequiredFields()
	fields[0] = enum.InvoiceCurrency
	assert.Equal(t, enum.InvoiceNumber, cfg.RequiredFields()[0])
	clone := cfg.Clone()
	require.NotEmpty(t, clone.RequiredFields())
	require.NotSame(t, &cfg.RequiredFields()[0], &clone.RequiredFields()[0])
}
