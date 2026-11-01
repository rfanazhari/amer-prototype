package accessquery_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// currentQueryCase names a current-session query outcome.
type currentQueryCase struct {
	name    string
	execute error
	wantErr bool
}

// TestCurrentSessionOperationRequiresBoundSession withholds CSRF material when the session is unbound.
func TestCurrentSessionOperationRequiresBoundSession(t *testing.T) {
	tests := []currentQueryCase{
		{name: "current session", execute: nil, wantErr: false},
		{name: "unbound session", execute: service.ErrSessionUnbound, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := mocks.NewCurrentSessionOperation(t)
			view := &dto.AdministrativeCurrentSession{
				Account:   dto.AdministrativeAccount{ID: testdata.ID, Role: "HR", Status: "ACTIVE"},
				Session:   dto.AdministrativeSession{ID: testdata.SessionID, ExpiresAt: testdata.Now()},
				CSRFToken: "signed-csrf-material",
			}
			if tt.wantErr {
				op.On("Execute", mock.Anything).Return((*dto.AdministrativeCurrentSession)(nil), tt.execute).Once()
				got, err := op.Execute(context.Background())
				require.ErrorIs(t, err, service.ErrSessionUnbound)
				assert.Nil(t, got)
				return
			}
			op.On("Execute", mock.Anything).Return(view, nil).Once()
			got, err := op.Execute(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "signed-csrf-material", got.CSRFToken)
			var _ accessquery.CurrentSessionOperation = op
		})
	}
}
