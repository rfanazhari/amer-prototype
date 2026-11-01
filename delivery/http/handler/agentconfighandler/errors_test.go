package agentconfighandler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// errorHTTPCase names one ACP classification, session or CSRF outcome.
type errorHTTPCase struct {
	name       string
	method     string
	path       string
	body       string
	csrf       string
	cookie     *http.Cookie
	locale     string
	cause      error
	logged     bool
	wantStatus int
	wantCode   string
	wantMsg    string
}

// TestMapsACPErrorsAndSessionBoundaries covers cookie, CSRF, ADMIN, candidate and §6.5 precedence.
func TestMapsACPErrorsAndSessionBoundaries(t *testing.T) {
	instancePath := "/admin/agent-instances/" + instanceID
	secret := errors.New("secret database diagnostic")
	tests := []errorHTTPCase{
		{name: "missing cookie", method: http.MethodGet, path: "/admin/agent-templates", wantStatus: http.StatusUnauthorized, wantCode: errs.ACPUnauthenticated},
		{name: "candidate cookie", method: http.MethodGet, path: "/admin/agent-templates", cookie: &http.Cookie{Name: "ia-candidate", Value: sessionTok}, wantStatus: http.StatusUnauthorized, wantCode: errs.ACPUnauthenticated},
		{name: "expired session", method: http.MethodGet, path: "/admin/agent-templates", cookie: adminCookie(false, sessionTok), cause: service.ErrSessionUnbound, wantStatus: http.StatusUnauthorized, wantCode: errs.ACPUnauthenticated},
		{name: "admin forbidden", method: http.MethodGet, path: "/admin/agent-templates", cookie: adminCookie(false, sessionTok), cause: service.ErrForbidden, wantStatus: http.StatusForbidden, wantCode: errs.ACPForbidden},
		{name: "missing csrf", method: http.MethodPost, path: "/admin/agent-instances", body: createHRBody, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusForbidden, wantCode: errs.ACPForbidden},
		{name: "invalid csrf", method: http.MethodPost, path: "/admin/agent-instances", body: createHRBody, csrf: "wrong", cookie: adminCookie(false, sessionTok), wantStatus: http.StatusForbidden, wantCode: errs.ACPForbidden},
		{name: "not found", method: http.MethodGet, path: instancePath, cookie: adminCookie(false, sessionTok), cause: agentconfigport.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: errs.NotFound},
		{name: "idempotency conflict", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), cause: service.ErrIdempotencyConflict, wantStatus: http.StatusConflict, wantCode: errs.ACPIdempotencyConflict},
		{name: "draft conflict", method: http.MethodPut, path: instancePath + "/draft", body: `{"expectedDraftVersion":1,"configuration":{}}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), cause: service.ErrDraftConflict, wantStatus: http.StatusConflict, wantCode: errs.ACPDraftConflict},
		{name: "validation conflict", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), cause: service.ErrValidationConflict, wantStatus: http.StatusConflict, wantCode: errs.ACPValidationConflict},
		{name: "publication blocked", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), cause: service.ErrPublicationBlocked, wantStatus: http.StatusUnprocessableEntity, wantCode: errs.ACPPublicationBlocked},
		{name: "dependency unavailable", method: http.MethodPost, path: instancePath + "/validations", body: `{"expectedDraftVersion":1}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), cause: service.ErrDependencyUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: errs.ACPDependencyUnavailable},
		{name: "publication uncertain", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), cause: &agentconfigport.CommitUncertainError{Cause: secret}, wantStatus: http.StatusServiceUnavailable, wantCode: errs.ACPPublicationUncertain},
		{name: "technical failure", method: http.MethodGet, path: instancePath, cookie: adminCookie(false, sessionTok), cause: errs.Technical{Cause: secret}, logged: true, wantStatus: http.StatusInternalServerError, wantCode: errs.InternalError},
		{name: "indonesian unauthenticated", method: http.MethodGet, path: "/admin/agent-templates", locale: "id", wantStatus: http.StatusUnauthorized, wantCode: errs.ACPUnauthenticated, wantMsg: "sesi administratif hilang atau tidak lagi valid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logged := false
			handler, catalogOp, create, instance, replace, validate, publish, _, csrf, sessions := newHandler(t, httpOrigin, false)
			handler.LogError = func(err error) { logged = true; require.ErrorIs(t, err, tt.cause) }
			if tt.cookie != nil && tt.cookie.Name == "ia-admin" {
				session := testdata.AdministrativeSessionRecord(t)
				if errors.Is(tt.cause, service.ErrSessionUnbound) && tt.method == http.MethodGet {
					catalogOp.On("Execute", mock.Anything).Return(nil, tt.cause).Once()
				} else {
					sessions.On("Resolve", mock.Anything).Return(session, testdata.AdministrativeAccountRecord(t), nil)
				}
			}
			if tt.csrf == csrfTok {
				csrf.On("Verify", service.CredentialAdministrative, testdata.SessionID, tt.csrf).Return(nil)
			}
			if tt.csrf != "" && tt.csrf != csrfTok && tt.cookie != nil && tt.cookie.Name == "ia-admin" {
				csrf.On("Verify", service.CredentialAdministrative, testdata.SessionID, tt.csrf).Return(service.ErrCSRFMismatch)
			}
			switch {
			case tt.path == "/admin/agent-templates" && tt.cause != nil && !errors.Is(tt.cause, service.ErrSessionUnbound):
				catalogOp.On("Execute", mock.Anything).Return(nil, tt.cause).Once()
			case tt.path == instancePath && tt.method == http.MethodGet && tt.cause != nil:
				instance.On("Execute", mock.Anything, instanceID).Return(nil, tt.cause).Once()
			case tt.path == "/admin/agent-instances" && tt.cause != nil:
				create.On("Execute", mock.Anything, mock.Anything).Return(nil, tt.cause).Once()
			case tt.path == instancePath+"/draft" && tt.cause != nil:
				replace.On("Execute", mock.Anything, mock.Anything).Return(nil, tt.cause).Once()
			case tt.path == instancePath+"/validations" && tt.cause != nil:
				validate.On("Execute", mock.Anything, mock.Anything).Return(nil, tt.cause).Once()
			case tt.path == instancePath+"/publications" && tt.cause != nil:
				publish.On("Execute", mock.Anything, mock.Anything).Return(nil, tt.cause).Once()
			}
			req := request(tt.method, tt.path, httpOrigin, tt.body, tt.csrf, tt.cookie, nil)
			if tt.path == instancePath+"/publications" {
				req.Header.Set("Idempotency-Key", publishKey)
			}
			if tt.locale != "" {
				req.Header.Set("Accept-Language", tt.locale)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			view := decodeEnvelope(t, rec)
			require.Equal(t, "error", view.Status)
			require.Equal(t, tt.wantCode, view.Error.Code)
			require.NotEqual(t, errs.IAUnauthenticated, view.Error.Code)
			require.NotEqual(t, errs.IAForbidden, view.Error.Code)
			require.NotContains(t, rec.Body.String(), "secret database diagnostic")
			require.Equal(t, tt.logged, logged)
			if tt.wantMsg != "" {
				require.Equal(t, tt.wantMsg, view.Error.Message)
			}
		})
	}
}
