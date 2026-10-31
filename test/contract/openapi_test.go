package contract_test

import (
	"context"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestOpenAPIContract validates the machine-readable endpoint specification and schema references.
func TestOpenAPIContract(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromFile("../../docs/api/openapi.yaml")
	require.NoError(t, err)
	require.NoError(t, document.Validate(context.Background()))
}
