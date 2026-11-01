package executionquery_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/executionquery"
	"github.com/stretchr/testify/assert"
)

// nopFind is a stub that satisfies FindOperation for compile-time interface checking only.
type nopFind struct{}

// Execute is the stub FindOperation implementation used only for compile-time interface checking.
func (n *nopFind) Execute(_ context.Context, executionID string) (*dto.ExecutionView, error) {
	_ = executionID
	return nil, nil
}

// nopFindByAssignment is a stub that satisfies FindByAssignmentOperation for compile-time interface checking only.
type nopFindByAssignment struct{}

// Execute is the stub FindByAssignmentOperation implementation used only for compile-time interface checking.
func (n *nopFindByAssignment) Execute(_ context.Context, assignmentID string) (*dto.ExecutionView, error) {
	_ = assignmentID
	return nil, nil
}

// TestQueryInterfacesDefined verifies that Find and FindByAssignment query interfaces are declared.
func TestQueryInterfacesDefined(t *testing.T) {
	var (
		_ executionquery.FindOperation              = (*nopFind)(nil)
		_ executionquery.FindByAssignmentOperation  = (*nopFindByAssignment)(nil)
	)
	assert.True(t, true, "both query interfaces are defined")
}
