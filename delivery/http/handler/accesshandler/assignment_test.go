package accesshandler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const assignmentCreatePath = "/admin/interview-assignments"

const assignmentCreateBody = `{"candidateId":"` + testdata.CandidateID + `","instanceId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8"}`

// assignmentHTTPCase names one interview-assignment create transport outcome.
type assignmentHTTPCase struct {
	name       string
	method     string
	path       string
	origin     string
	body       string
	csrf       string
	cookie     *http.Cookie
	callCreate bool
	opErr      error
	wantStatus int
	wantCode   string
}

// newAssignmentHandler wires the generated create-assignment operation for one HTTP scenario.
func newAssignmentHandler(t *testing.T) (accesshandler.Handler, *mocks.CreateAssignmentOperation, *mocks.CSRFTokens, *mocks.CurrentAdministrativeSession) {
	t.Helper()
	create := mocks.NewCreateAssignmentOperation(t)
	csrf := mocks.NewCSRFTokens(t)
	admin := mocks.NewCurrentAdministrativeSession(t)
	return accesshandler.Handler{
		CreateAssignment: create,
		CSRF:             csrf,
		Sessions:         admin,
		Origin:           httpOrigin,
		SecureCookie:     false,
		Limiter:          &accesshandler.LoginLimiter{},
		ExchangeLimiter:  &accesshandler.LoginLimiter{},
	}, create, csrf, admin
}

// TestCreateInterviewAssignmentHTTP covers Origin, CSRF, ADMIN, candidate cookie and success mapping.
func TestCreateInterviewAssignmentHTTP(t *testing.T) {
	created := &dto.Assignment{
		ID: testdata.AssignmentID, CandidateID: testdata.CandidateID,
		InstanceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8", PublishedRevisionID: testdata.AssignmentFacts(t).RevisionID,
	}
	tests := []assignmentHTTPCase{
		{name: "hr creates assignment", method: http.MethodPost, path: assignmentCreatePath, origin: httpOrigin, body: assignmentCreateBody, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callCreate: true, wantStatus: http.StatusCreated},
		{name: "unknown field", method: http.MethodPost, path: assignmentCreatePath, origin: httpOrigin, body: `{"candidateId":"` + testdata.CandidateID + `","instanceId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8","publishedRevisionId":"x"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "missing csrf", method: http.MethodPost, path: assignmentCreatePath, origin: httpOrigin, body: assignmentCreateBody, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusForbidden, wantCode: errs.IAForbidden},
		{name: "candidate cookie", method: http.MethodPost, path: assignmentCreatePath, origin: httpOrigin, body: assignmentCreateBody, csrf: csrfTok, cookie: candidateCookie(false, sessionTok), wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "admin forbidden", method: http.MethodPost, path: assignmentCreatePath, origin: httpOrigin, body: assignmentCreateBody, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callCreate: true, opErr: service.ErrForbidden, wantStatus: http.StatusForbidden, wantCode: errs.IAForbidden},
		{name: "unpublished instance", method: http.MethodPost, path: assignmentCreatePath, origin: httpOrigin, body: assignmentCreateBody, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callCreate: true, opErr: agentbindingport.ErrNoActiveRevision, wantStatus: http.StatusConflict, wantCode: errs.IANoActiveRevision},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, create, csrf, sessions := newAssignmentHandler(t)
			if tt.cookie != nil && tt.cookie.Name == "ia-admin" {
				sessions.On("Resolve", mock.Anything).Return(testdata.AdministrativeSessionRecord(t), testdata.AdministrativeAccountRecord(t), nil).Maybe()
			}
			if tt.csrf != "" {
				csrf.On("Verify", service.CredentialAdministrative, testdata.SessionID, tt.csrf).Return(nil).Maybe()
			}
			if tt.callCreate {
				create.On("Execute", mock.Anything, testdata.CandidateID, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8").Return(created, tt.opErr).Maybe()
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
			var data map[string]string
			require.NoError(t, json.Unmarshal(view.Data, &data))
			require.Equal(t, testdata.AssignmentID, data["id"])
			require.Equal(t, testdata.CandidateID, data["candidateId"])
			require.Equal(t, created.InstanceID, data["instanceId"])
			require.Equal(t, created.PublishedRevisionID, data["publishedRevisionId"])
			require.NotContains(t, data, "createdAt")
			require.NotContains(t, data, "updatedAt")
		})
	}
}
