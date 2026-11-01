package accessusecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// loginOperationCase names a login operation classified outcome.
type loginOperationCase struct {
	name    string
	execute error
	wantErr bool
}

// TestLoginOperationDoesNotReturnSessionOnProofFailure keeps login metadata absent when authentication fails.
func TestLoginOperationDoesNotReturnSessionOnProofFailure(t *testing.T) {
	tests := []loginOperationCase{
		{name: "committed login", execute: nil, wantErr: false},
		{name: "proof failure", execute: service.ErrAuthenticationFailed, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := mocks.NewLoginOperation(t)
			result := &dto.AdministrativeLogin{
				Account: dto.AdministrativeAccount{ID: testdata.ID, Email: "sinta@example.com", Role: "HR"},
				Session: dto.AdministrativeSession{ID: testdata.SessionID, ExpiresAt: testdata.Now().Add(2 * time.Hour)},
			}
			if tt.wantErr {
				op.On("Execute", mock.Anything, "sinta@example.com", "wrong").Return((*dto.AdministrativeLogin)(nil), tt.execute).Once()
				got, err := op.Execute(context.Background(), "sinta@example.com", "wrong")
				require.ErrorIs(t, err, service.ErrAuthenticationFailed)
				assert.Nil(t, got)
				return
			}
			op.On("Execute", mock.Anything, "sinta@example.com", "correct-horse").Return(result, nil).Once()
			got, err := op.Execute(context.Background(), "sinta@example.com", "correct-horse")
			require.NoError(t, err)
			assert.Equal(t, testdata.ID, got.Account.ID)
			assert.Equal(t, testdata.SessionID, got.Session.ID)
			var _ accessusecase.LoginOperation = op
		})
	}
}
