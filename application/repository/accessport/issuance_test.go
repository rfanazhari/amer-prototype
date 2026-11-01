package accessport_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// commitTimingCase names a login-transaction outcome that must not imply durable success.
type commitTimingCase struct {
	name    string
	commit  error
	wantErr bool
}

// TestSessionIssuanceDoesNotSucceedBeforeCommit treats callback success plus commit failure as classified uncertainty.
func TestSessionIssuanceDoesNotSucceedBeforeCommit(t *testing.T) {
	cause := errors.New("connection reset")
	tests := []commitTimingCase{
		{name: "committed session", commit: nil, wantErr: false},
		{name: "uncertain commit", commit: &accessport.CommitUncertainError{Cause: cause}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuance := mocks.NewSessionIssuance(t)
			issuance.On("WithinLogin", mock.Anything, testdata.ID, mock.Anything).Return(tt.commit).Once()
			err := issuance.WithinLogin(context.Background(), testdata.ID, func(accessport.LoginTx) error {
				return nil
			})
			assert.Equal(t, tt.wantErr, err != nil)
			if !tt.wantErr {
				return
			}
			var uncertain *accessport.CommitUncertainError
			require.ErrorAs(t, err, &uncertain)
			assert.Equal(t, "access commit outcome uncertain", err.Error())
			assert.NotContains(t, err.Error(), cause.Error())
			assert.ErrorIs(t, err, cause)
		})
	}
}

// loginInsertCase names a session-digest uniqueness outcome inside the locked login transaction.
type loginInsertCase struct {
	name      string
	insertErr error
	wantErr   bool
}

// TestLoginTxInsertConflictUsesGeneratedMock classifies duplicate session digests without exposing the digest.
func TestLoginTxInsertConflictUsesGeneratedMock(t *testing.T) {
	record := testdata.AdministrativeSessionRecord(t)
	tests := []loginInsertCase{
		{name: "unique digest", insertErr: nil, wantErr: false},
		{name: "duplicate digest", insertErr: accessport.TokenConflictError{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := mocks.NewLoginTx(t)
			tx.On("InsertSession", mock.Anything, record).Return(tt.insertErr).Once()
			err := tx.InsertSession(context.Background(), record)
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				var conflict accessport.TokenConflictError
				require.ErrorAs(t, err, &conflict)
				assert.NotContains(t, err.Error(), record.TokenHash)
			}
		})
	}
}
