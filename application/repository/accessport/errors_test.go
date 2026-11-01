package accessport_test

import (
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/stretchr/testify/assert"
)

// conflictCase names a classified persistence conflict and its safe public message.
type conflictCase struct {
	name    string
	err     error
	message string
	wantErr bool
}

// TestAccessPersistenceConflictsHideIdentifiers keeps uniqueness failures free of emails, IDs and digests.
func TestAccessPersistenceConflictsHideIdentifiers(t *testing.T) {
	tests := []conflictCase{
		{name: "duplicate canonical email", err: accessport.AccountConflictError{}, message: "administrative account conflict", wantErr: true},
		{name: "duplicate session digest", err: accessport.TokenConflictError{}, message: "administrative session token conflict", wantErr: true},
		{name: "missing access row", err: accessport.ErrNotFound, message: "administrative access record not found", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantErr, tt.err != nil)
			assert.Equal(t, tt.message, tt.err.Error())
			assert.NotContains(t, tt.err.Error(), "@")
			assert.False(t, errors.Is(tt.err, errors.New(tt.message)))
		})
	}
}
