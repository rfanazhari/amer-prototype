package accessauthority_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// authorityCase names one real ACP authority adapter outcome.
type authorityCase struct {
	name      string
	account   accessport.AccountRecord
	session   accessport.SessionRecord
	principal service.AdminPrincipal
	operation string
	resource  string
	resolve   error
	now       time.Time
	wantAuth  error
	wantPrinc error
}

// TestAuthorityRechecksCurrentHRAndStableSubject covers HR allow, ADMIN deny, replay and unauthenticated paths.
func TestAuthorityRechecksCurrentHRAndStableSubject(t *testing.T) {
	session := testdata.AdministrativeSessionRecord(t)
	admin := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	otherHR := testdata.AdministrativeAccountRecord(t)
	otherHR.Role = enum.HR
	otherHR.ID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	disabled := testdata.AdministrativeAccountRecord(t)
	disabled.Role = enum.HR
	disabled.Status = enum.Disabled
	revokedAt := testdata.Now().Add(time.Minute)
	revoked := testdata.AdministrativeSessionRecord(t)
	revoked.RevokedAt = &revokedAt
	tests := []authorityCase{
		{name: "current hr allowed", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.AgentConfigurationCreate, now: testdata.Now().Add(time.Minute)},
		{name: "hr reads another hr resource", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.AgentConfigurationRead, resource: otherHR.ID, now: testdata.Now().Add(time.Minute)},
		{name: "hr replay rechecks current authority", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.AgentConfigurationReplay, now: testdata.Now().Add(time.Minute)},
		{name: "valid admin forbidden", account: admin, session: session, principal: service.AdminPrincipal{SubjectID: admin.ID}, operation: service.AgentConfigurationPublish, now: testdata.Now().Add(time.Minute), wantAuth: service.ErrForbidden},
		{name: "unknown operation denied", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: "delete-all", now: testdata.Now().Add(time.Minute), wantAuth: service.ErrForbidden},
		{name: "expired session unauthenticated", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.AgentConfigurationRead, now: testdata.Now().Add(2 * time.Hour), wantPrinc: service.ErrSessionUnbound, wantAuth: service.ErrSessionUnbound},
		{name: "revoked session unauthenticated", account: hr, session: revoked, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.AgentConfigurationRead, now: testdata.Now().Add(2 * time.Minute), wantPrinc: service.ErrSessionUnbound, wantAuth: service.ErrSessionUnbound},
		{name: "disabled account unauthenticated", account: disabled, session: session, principal: service.AdminPrincipal{SubjectID: disabled.ID}, operation: service.AgentConfigurationRead, now: testdata.Now().Add(time.Minute), wantPrinc: service.ErrSessionUnbound, wantAuth: service.ErrSessionUnbound},
		{name: "missing session unauthenticated", resolve: service.ErrSessionUnbound, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.AgentConfigurationRead, wantPrinc: service.ErrSessionUnbound, wantAuth: service.ErrSessionUnbound},
		{name: "caller principal mismatch unbound", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: "ffffffff-ffff-4fff-8fff-ffffffffffff"}, operation: service.AgentConfigurationRead, now: testdata.Now().Add(time.Minute), wantAuth: service.ErrSessionUnbound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := mocks.NewCurrentAdministrativeSession(t)
			clock := mocks.NewClock(t)
			if tt.resolve != nil {
				resolver.On("Resolve", mock.Anything).Return(accessport.SessionRecord{}, accessport.AccountRecord{}, tt.resolve)
			} else {
				resolver.On("Resolve", mock.Anything).Return(tt.session, tt.account, nil)
				clock.On("Now").Return(tt.now)
			}
			authority := accessauthority.Authority{Sessions: resolver, Clock: clock}
			var port service.AgentConfigurationAuthority = authority
			got, err := port.Principal(context.Background())
			if tt.wantPrinc != nil {
				require.ErrorIs(t, err, tt.wantPrinc)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.account.ID, got.SubjectID)
				assert.NotEqual(t, tt.session.ID, got.SubjectID)
			}
			err = port.Authorize(context.Background(), tt.principal, tt.operation, tt.resource)
			if tt.wantAuth != nil {
				require.ErrorIs(t, err, tt.wantAuth)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestAuthorityStableSubjectAcrossSessions uses the account UUID rather than the session identity.
func TestAuthorityStableSubjectAcrossSessions(t *testing.T) {
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	first := testdata.AdministrativeSessionRecord(t)
	second := testdata.AdministrativeSessionRecord(t)
	second.ID = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	resolver := mocks.NewCurrentAdministrativeSession(t)
	clock := mocks.NewClock(t)
	clock.On("Now").Return(testdata.Now().Add(time.Minute))
	resolver.On("Resolve", mock.Anything).Return(first, hr, nil).Once()
	resolver.On("Resolve", mock.Anything).Return(second, hr, nil).Once()
	authority := accessauthority.Authority{Sessions: resolver, Clock: clock}
	firstPrincipal, err := authority.Principal(context.Background())
	require.NoError(t, err)
	secondPrincipal, err := authority.Principal(context.Background())
	require.NoError(t, err)
	assert.Equal(t, hr.ID, firstPrincipal.SubjectID)
	assert.Equal(t, firstPrincipal.SubjectID, secondPrincipal.SubjectID)
	assert.NotEqual(t, first.ID, second.ID)
}
