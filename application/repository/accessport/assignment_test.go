package accessport_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const assignmentID = "5b1c0e22-7d4f-4a91-8c33-2b3c4d5e6f70"

// assignmentReadinessCase names assignment-first collaboration readiness.
type assignmentReadinessCase struct {
	name    string
	err     error
	wantErr bool
}

// TestAssignmentAccessCollaborationIsQ2Dependent keeps candidate assignment locking unready until Interview Q2.
func TestAssignmentAccessCollaborationIsQ2Dependent(t *testing.T) {
	tests := []assignmentReadinessCase{
		{name: "q2 owner missing", err: accessport.ErrAssignmentAccessNotReady, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			access := mocks.NewAssignmentAccess(t)
			access.On("WithAssignment", mock.Anything, assignmentID, mock.Anything).Return(tt.err).Once()
			err := access.WithAssignment(context.Background(), assignmentID, func(accessport.AssignmentTx) error {
				t.Fatal("callback must not run when assignment access is not ready")
				return nil
			})
			assert.Equal(t, tt.wantErr, err != nil)
			require.ErrorIs(t, err, accessport.ErrAssignmentAccessNotReady)
			assert.Equal(t, "assignment access collaboration is not ready until interview Q2", err.Error())
		})
	}
}
