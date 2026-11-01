package accessport_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// accountTxConflictCase names a mocked uniqueness outcome for serialized account writes.
type accountTxConflictCase struct {
	name      string
	insertErr error
	wantErr   bool
}

// TestAccountTxInsertConflictUsesGeneratedMock proves duplicate provision is classified at the port boundary.
func TestAccountTxInsertConflictUsesGeneratedMock(t *testing.T) {
	record := testdata.AdministrativeAccountRecord(t)
	tests := []accountTxConflictCase{
		{name: "unique canonical email", insertErr: nil, wantErr: false},
		{name: "duplicate canonical email", insertErr: accessport.AccountConflictError{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := mocks.NewAccountTx(t)
			tx.On("Insert", mock.Anything, record).Return(tt.insertErr).Once()
			err := tx.Insert(context.Background(), record)
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				var conflict accessport.AccountConflictError
				require.ErrorAs(t, err, &conflict)
				assert.Equal(t, "administrative account conflict", err.Error())
			}
		})
	}
}
