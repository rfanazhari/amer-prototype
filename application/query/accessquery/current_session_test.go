package accessquery_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// currentSessionImplCase names one current-session query outcome against generated ports.
type currentSessionImplCase struct {
	name     string
	session  accessport.SessionRecord
	account  accessport.AccountRecord
	resolve  error
	now      time.Time
	wantErr  error
	wantCSRF bool
}

// TestCurrentSessionReturnsStableSubjectAndCSRF covers current HR, exact expiry and disabled denial.
func TestCurrentSessionReturnsStableSubjectAndCSRF(t *testing.T) {
	session := testdata.AdministrativeSessionRecord(t)
	account := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	disabled := testdata.AdministrativeAccountRecord(t)
	disabled.Status = enum.Disabled
	tests := []currentSessionImplCase{
		{name: "current hr session", session: session, account: hr, now: testdata.Now().Add(time.Minute), wantCSRF: true},
		{name: "stable admin subject", session: session, account: account, now: testdata.Now().Add(time.Minute), wantCSRF: true},
		{name: "exact expiry refused", session: session, account: hr, now: testdata.Now().Add(2 * time.Hour), wantErr: service.ErrSessionUnbound},
		{name: "disabled account", session: session, account: disabled, now: testdata.Now().Add(time.Minute), wantErr: service.ErrSessionUnbound},
		{name: "unbound session", resolve: service.ErrSessionUnbound, wantErr: service.ErrSessionUnbound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := mocks.NewCurrentAdministrativeSession(t)
			csrf := mocks.NewCSRFTokens(t)
			clock := mocks.NewClock(t)
			if tt.resolve != nil {
				current.On("Resolve", mock.Anything).Return(accessport.SessionRecord{}, accessport.AccountRecord{}, tt.resolve).Once()
			} else {
				current.On("Resolve", mock.Anything).Return(tt.session, tt.account, nil).Once()
				clock.On("Now").Return(tt.now)
			}
			if tt.wantCSRF {
				csrf.On("Issue", service.CredentialAdministrative, tt.session.ID).Return("signed-csrf-material", nil).Once()
			}
			got, err := (accessquery.CurrentSession{Current: current, CSRF: csrf, Clock: clock}).Execute(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.account.ID, got.Account.ID)
			assert.Equal(t, tt.account.Role.String(), got.Account.Role)
			assert.Equal(t, tt.session.ID, got.Session.ID)
			assert.Equal(t, tt.session.ExpiresAt, got.Session.ExpiresAt)
			assert.Equal(t, "signed-csrf-material", got.CSRFToken)
		})
	}
}
