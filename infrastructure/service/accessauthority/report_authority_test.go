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

// reportAssignmentID is the assignment resource used by the report authority cases.
const reportAssignmentID = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"

// reportAuthorityCase names one interview report authority outcome on the shared adapter.
type reportAuthorityCase struct {
	name      string
	account   accessport.AccountRecord
	session   accessport.SessionRecord
	principal service.AdminPrincipal
	operation string
	resolve   error
	now       time.Time
	wantAuth  error
	wantPrinc error
}

// TestReportAuthorityRechecksCurrentHRForReportOperations covers HR allow, ADMIN deny and every unauthenticated path.
func TestReportAuthorityRechecksCurrentHRForReportOperations(t *testing.T) {
	session := testdata.AdministrativeSessionRecord(t)
	admin := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	disabled := testdata.AdministrativeAccountRecord(t)
	disabled.Role = enum.HR
	disabled.Status = enum.Disabled
	revokedAt := testdata.Now().Add(time.Minute)
	revoked := testdata.AdministrativeSessionRecord(t)
	revoked.RevokedAt = &revokedAt
	tests := []reportAuthorityCase{
		{name: "current hr reads", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.InterviewReportRead, now: testdata.Now().Add(time.Minute)},
		{name: "current hr regenerates", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.InterviewReportRegenerate, now: testdata.Now().Add(time.Minute)},
		{name: "valid admin forbidden to read", account: admin, session: session, principal: service.AdminPrincipal{SubjectID: admin.ID}, operation: service.InterviewReportRead, now: testdata.Now().Add(time.Minute), wantAuth: service.ErrForbidden},
		{name: "valid admin forbidden to regenerate", account: admin, session: session, principal: service.AdminPrincipal{SubjectID: admin.ID}, operation: service.InterviewReportRegenerate, now: testdata.Now().Add(time.Minute), wantAuth: service.ErrForbidden},
		{name: "configuration operation is not a report operation", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.AgentConfigurationPublish, now: testdata.Now().Add(time.Minute), wantAuth: service.ErrForbidden},
		{name: "unknown operation denied", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: "delete-all", now: testdata.Now().Add(time.Minute), wantAuth: service.ErrForbidden},
		{name: "expired session unauthenticated", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.InterviewReportRead, now: testdata.Now().Add(2 * time.Hour), wantPrinc: service.ErrSessionUnbound, wantAuth: service.ErrSessionUnbound},
		{name: "revoked session unauthenticated", account: hr, session: revoked, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.InterviewReportRead, now: testdata.Now().Add(2 * time.Minute), wantPrinc: service.ErrSessionUnbound, wantAuth: service.ErrSessionUnbound},
		{name: "disabled account unauthenticated", account: disabled, session: session, principal: service.AdminPrincipal{SubjectID: disabled.ID}, operation: service.InterviewReportRead, now: testdata.Now().Add(time.Minute), wantPrinc: service.ErrSessionUnbound, wantAuth: service.ErrSessionUnbound},
		{name: "missing or candidate session unbound", resolve: service.ErrSessionUnbound, principal: service.AdminPrincipal{SubjectID: hr.ID}, operation: service.InterviewReportRead, wantPrinc: service.ErrSessionUnbound, wantAuth: service.ErrSessionUnbound},
		{name: "caller principal mismatch unbound", account: hr, session: session, principal: service.AdminPrincipal{SubjectID: "ffffffff-ffff-4fff-8fff-ffffffffffff"}, operation: service.InterviewReportRead, now: testdata.Now().Add(time.Minute), wantAuth: service.ErrSessionUnbound},
		{name: "empty principal unbound", account: hr, session: session, principal: service.AdminPrincipal{}, operation: service.InterviewReportRead, now: testdata.Now().Add(time.Minute), wantAuth: service.ErrSessionUnbound},
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
			var port service.InterviewReportAuthority = accessauthority.ReportAuthority(accessauthority.Authority{Sessions: resolver, Clock: clock})
			got, err := port.Principal(context.Background())
			if tt.wantPrinc != nil {
				require.ErrorIs(t, err, tt.wantPrinc)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.account.ID, got.SubjectID)
				assert.NotEqual(t, tt.session.ID, got.SubjectID)
			}
			err = port.Authorize(context.Background(), tt.principal, tt.operation, reportAssignmentID)
			if tt.wantAuth != nil {
				require.ErrorIs(t, err, tt.wantAuth)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestAuthorityConfigurationAuthorizeIgnoresReportOperations proves the configuration catalog is unchanged.
func TestAuthorityConfigurationAuthorizeIgnoresReportOperations(t *testing.T) {
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	resolver := mocks.NewCurrentAdministrativeSession(t)
	clock := mocks.NewClock(t)
	resolver.On("Resolve", mock.Anything).Return(testdata.AdministrativeSessionRecord(t), hr, nil)
	clock.On("Now").Return(testdata.Now().Add(time.Minute))
	var port service.AgentConfigurationAuthority = accessauthority.Authority{Sessions: resolver, Clock: clock}
	err := port.Authorize(context.Background(), service.AdminPrincipal{SubjectID: hr.ID}, service.InterviewReportRegenerate, "")
	require.ErrorIs(t, err, service.ErrForbidden)
	require.NoError(t, port.Authorize(context.Background(), service.AdminPrincipal{SubjectID: hr.ID}, service.AgentConfigurationRead, ""))
}
