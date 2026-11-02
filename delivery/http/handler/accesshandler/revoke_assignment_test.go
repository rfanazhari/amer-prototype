package accesshandler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// revocationPath is the HR assignment access revocation route for the fixture assignment.
const revocationPath = "/admin/interview-assignments/" + testdata.AssignmentID + "/revocation"

// revokeHTTPCase names one assignment revocation transport outcome.
type revokeHTTPCase struct {
	name       string
	method     string
	path       string
	origin     string
	body       string
	csrf       string
	cookie     *http.Cookie
	callRevoke bool
	opErr      error
	wantStatus int
	wantCode   string
}

// TestRevokeAssignmentAccessHTTP covers the revocation route's guards, error mapping and success body.
func TestRevokeAssignmentAccessHTTP(t *testing.T) {
	revoked := &dto.AssignmentRevocation{ID: testdata.AssignmentID, AccessRevokedAt: testdata.Now()}
	tests := []revokeHTTPCase{
		{name: "hr revokes with no body", method: http.MethodPost, path: revocationPath, origin: httpOrigin, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callRevoke: true, wantStatus: http.StatusOK},
		{name: "hr revokes with empty object", method: http.MethodPost, path: revocationPath, origin: httpOrigin, body: `{}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callRevoke: true, wantStatus: http.StatusOK},
		{name: "unknown field is rejected before the operation", method: http.MethodPost, path: revocationPath, origin: httpOrigin, body: `{"reason":"x"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "missing csrf", method: http.MethodPost, path: revocationPath, origin: httpOrigin, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusForbidden, wantCode: errs.IAForbidden},
		{name: "candidate cookie", method: http.MethodPost, path: revocationPath, origin: httpOrigin, csrf: csrfTok, cookie: candidateCookie(false, sessionTok), wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "missing session", method: http.MethodPost, path: revocationPath, origin: httpOrigin, csrf: csrfTok, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "admin forbidden", method: http.MethodPost, path: revocationPath, origin: httpOrigin, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callRevoke: true, opErr: service.ErrForbidden, wantStatus: http.StatusForbidden, wantCode: errs.IAForbidden},
		{name: "malformed id", method: http.MethodPost, path: "/admin/interview-assignments/not-a-uuid/revocation", origin: httpOrigin, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callRevoke: true, opErr: service.ErrInvalidInput, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "unknown assignment", method: http.MethodPost, path: revocationPath, origin: httpOrigin, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callRevoke: true, opErr: accessport.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: errs.NotFound},
		{name: "cancellation incomplete", method: http.MethodPost, path: revocationPath, origin: httpOrigin, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callRevoke: true, opErr: service.ErrCancellationIncomplete, wantStatus: http.StatusServiceUnavailable, wantCode: errs.IADependencyUnavailable},
		{name: "get is not allowed", method: http.MethodGet, path: revocationPath, origin: httpOrigin, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revoke := mocks.NewRevokeAssignmentAccessOperation(t)
			csrf := mocks.NewCSRFTokens(t)
			sessions := mocks.NewCurrentAdministrativeSession(t)
			handler := accesshandler.Handler{
				RevokeAssignment: revoke, CSRF: csrf, Sessions: sessions, Origin: httpOrigin,
				Limiter: &accesshandler.LoginLimiter{}, ExchangeLimiter: &accesshandler.LoginLimiter{},
			}
			if tt.cookie != nil && tt.cookie.Name == "ia-admin" {
				sessions.On("Resolve", mock.Anything).Return(testdata.AdministrativeSessionRecord(t), testdata.AdministrativeAccountRecord(t), nil).Maybe()
			}
			if tt.csrf != "" {
				csrf.On("Verify", service.CredentialAdministrative, testdata.SessionID, tt.csrf).Return(nil).Maybe()
			}
			if tt.callRevoke {
				var result *dto.AssignmentRevocation
				if tt.opErr == nil {
					result = revoked
				}
				revoke.On("Execute", mock.Anything, mock.Anything).Return(result, tt.opErr).Once()
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request(tt.method, tt.path, tt.origin, tt.body, tt.csrf, tt.cookie))
			require.Equal(t, tt.wantStatus, rec.Code)
			view := decodeEnvelope(t, rec)
			if tt.wantCode != "" {
				require.Equal(t, "error", view.Status)
				require.Equal(t, tt.wantCode, view.Error.Code)
				return
			}
			require.Equal(t, "success", view.Status)
			var data map[string]any
			require.NoError(t, json.Unmarshal(view.Data, &data))
			require.Equal(t, map[string]any{"id": testdata.AssignmentID, "accessRevokedAt": "2026-01-01T00:00:00.000Z"}, data)
		})
	}
}
