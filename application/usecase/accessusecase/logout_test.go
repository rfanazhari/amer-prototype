package accessusecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// logoutCase names one current-session revocation outcome.
type logoutCase struct {
	name      string
	session   accessport.SessionRecord
	account   accessport.AccountRecord
	resolve   error
	updateErr error
	now       time.Time
	wantErr   error
	wantWrite bool
}

// TestLogoutRevokesOnlyCurrentBoundSession withholds success on expiry, disablement and uncertain writes.
func TestLogoutRevokesOnlyCurrentBoundSession(t *testing.T) {
	session := testdata.AdministrativeSessionRecord(t)
	account := testdata.AdministrativeAccountRecord(t)
	expired := testdata.AdministrativeSessionRecord(t)
	disabled := testdata.AdministrativeAccountRecord(t)
	disabled.Status = enum.Disabled
	uncertain := &accessport.CommitUncertainError{Cause: context.DeadlineExceeded}
	tests := []logoutCase{
		{name: "current session revoked", session: session, account: account, now: testdata.Now().Add(time.Minute), wantWrite: true},
		{name: "expired session", session: expired, account: account, now: testdata.Now().Add(2 * time.Hour), wantErr: service.ErrSessionUnbound},
		{name: "disabled account", session: session, account: disabled, now: testdata.Now().Add(time.Minute), wantErr: service.ErrSessionUnbound},
		{name: "unbound session", resolve: service.ErrSessionUnbound, wantErr: service.ErrSessionUnbound},
		{name: "uncertain commit is not success", session: session, account: account, now: testdata.Now().Add(time.Minute), updateErr: uncertain, wantErr: uncertain, wantWrite: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := mocks.NewCurrentAdministrativeSession(t)
			sessions := mocks.NewSessions(t)
			clock := mocks.NewClock(t)
			if tt.resolve != nil {
				current.On("Resolve", mock.Anything).Return(accessport.SessionRecord{}, accessport.AccountRecord{}, tt.resolve).Once()
			} else {
				current.On("Resolve", mock.Anything).Return(tt.session, tt.account, nil).Once()
				clock.On("Now").Return(tt.now)
			}
			if tt.wantWrite {
				sessions.On("UpdateRevocation", mock.Anything, tt.session.ID, mock.MatchedBy(func(revoked *time.Time) bool {
					return revoked != nil && revoked.Equal(tt.now)
				})).Return(tt.updateErr).Once()
			}
			err := (accessusecase.Logout{Current: current, Sessions: sessions, Clock: clock}).Execute(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.NotContains(t, err.Error(), testdata.TokenHash)
				return
			}
			require.NoError(t, err)
		})
	}
}
