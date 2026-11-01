package accesshandler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

const invitationTok = "invitation-secret"

// candidateHTTPCase names one candidate identity, invitation or session transport outcome.
type candidateHTTPCase struct {
	name         string
	method       string
	path         string
	origin       string
	body         string
	csrf         string
	cookie       *http.Cookie
	secure       bool
	callCreate   bool
	callRead     bool
	callIssue    bool
	callExchange bool
	callSession  bool
	opErr        error
	wantStatus   int
	wantCode     string
	wantCookie   string
}

// newCandidateHandler wires generated candidate operation mocks for one HTTP scenario.
func newCandidateHandler(t *testing.T, origin string, secure bool) (accesshandler.Handler, *mocks.CreateCandidateOperation, *mocks.CandidateOperation, *mocks.IssueInvitationOperation, *mocks.ExchangeInvitationOperation, *mocks.CandidateSessionOperation, *mocks.CSRFTokens, *mocks.CurrentAdministrativeSession) {
	t.Helper()
	create := mocks.NewCreateCandidateOperation(t)
	read := mocks.NewCandidateOperation(t)
	issue := mocks.NewIssueInvitationOperation(t)
	exchange := mocks.NewExchangeInvitationOperation(t)
	session := mocks.NewCandidateSessionOperation(t)
	csrf := mocks.NewCSRFTokens(t)
	admin := mocks.NewCurrentAdministrativeSession(t)
	return accesshandler.Handler{
		CreateCandidate:  create,
		Candidate:        read,
		IssueInvitation:  issue,
		Exchange:         exchange,
		CandidateSession: session,
		CSRF:             csrf,
		Sessions:         admin,
		Origin:           origin,
		SecureCookie:     secure,
		Limiter:          &accesshandler.LoginLimiter{},
		ExchangeLimiter:  &accesshandler.LoginLimiter{},
	}, create, read, issue, exchange, session, csrf, admin
}

// candidateCookie returns the configured candidate session cookie name for an origin.
func candidateCookie(secure bool, value string) *http.Cookie {
	name := "ia-candidate"
	if secure {
		name = "__Host-ia-candidate"
	}
	return &http.Cookie{Name: name, Value: value}
}

// TestCandidateIdentityAndInvitationHTTP covers HR candidate create/read and invitation issue transport.
func TestCandidateIdentityAndInvitationHTTP(t *testing.T) {
	candidate := dto.FromCandidate(testdata.CandidateRecord(t))
	issued := &dto.IssuedInvitation{
		ID: testdata.InvitationID, AssignmentID: testdata.AssignmentID,
		ExpiresAt: testdata.InvitationRecord(t).ExpiresAt, Token: invitationTok,
	}
	tests := []candidateHTTPCase{
		{name: "create candidate", method: http.MethodPost, path: "/admin/candidates", origin: httpOrigin, body: `{"name":" Andi Pratama ","email":"Andi@Example.com"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callCreate: true, wantStatus: http.StatusCreated},
		{name: "create unknown field", method: http.MethodPost, path: "/admin/candidates", origin: httpOrigin, body: `{"name":"Andi","email":"andi@example.com","role":"HR"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "create missing csrf", method: http.MethodPost, path: "/admin/candidates", origin: httpOrigin, body: `{"name":"Andi","email":"andi@example.com"}`, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusForbidden, wantCode: errs.IAForbidden},
		{name: "read candidate", method: http.MethodGet, path: "/admin/candidates/" + testdata.CandidateID, origin: httpOrigin, cookie: adminCookie(false, sessionTok), callRead: true, wantStatus: http.StatusOK},
		{name: "read unknown after auth", method: http.MethodGet, path: "/admin/candidates/" + testdata.CandidateID, origin: httpOrigin, cookie: adminCookie(false, sessionTok), callRead: true, opErr: accessport.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: errs.NotFound},
		{name: "issue invitation", method: http.MethodPost, path: "/admin/interview-assignments/" + testdata.AssignmentID + "/invitations", origin: httpOrigin, body: `{}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callIssue: true, wantStatus: http.StatusCreated},
		{name: "issue empty body", method: http.MethodPost, path: "/admin/interview-assignments/" + testdata.AssignmentID + "/invitations", origin: httpOrigin, csrf: csrfTok, cookie: adminCookie(false, sessionTok), callIssue: true, wantStatus: http.StatusCreated},
		{name: "admin cookie required", method: http.MethodGet, path: "/admin/candidates/" + testdata.CandidateID, origin: httpOrigin, cookie: candidateCookie(false, sessionTok), wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, create, read, issue, _, _, csrf, sessions := newCandidateHandler(t, httpOrigin, false)
			if tt.cookie != nil && tt.cookie.Name == "ia-admin" {
				sessions.On("Resolve", mock.Anything).Return(testdata.AdministrativeSessionRecord(t), testdata.AdministrativeAccountRecord(t), nil).Maybe()
			}
			if tt.csrf != "" {
				csrf.On("Verify", service.CredentialAdministrative, testdata.SessionID, tt.csrf).Return(nil).Maybe()
			}
			if tt.callCreate {
				create.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return(&candidate, tt.opErr).Maybe()
			}
			if tt.callRead {
				read.On("Execute", mock.Anything, testdata.CandidateID).Return(&candidate, tt.opErr).Maybe()
			}
			if tt.callIssue {
				issue.On("Execute", mock.Anything, testdata.AssignmentID).Return(issued, tt.opErr).Maybe()
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
			if tt.callIssue {
				require.Contains(t, rec.Body.String(), invitationTok)
			}
		})
	}
}

// TestCandidateExchangeAndSessionHTTP covers exchange cookies, GET refusal and current-session recovery.
func TestCandidateExchangeAndSessionHTTP(t *testing.T) {
	access := &dto.CandidateAccess{
		Candidate:    dto.FromCandidate(testdata.CandidateRecord(t)),
		AssignmentID: testdata.AssignmentID,
		Session:      dto.AdministrativeSession{ID: testdata.CandidateSessionID, ExpiresAt: testdata.CandidateSessionRecord(t).ExpiresAt},
		SessionToken: "candidate-session-secret",
		CSRFToken:    csrfTok,
	}
	tests := []candidateHTTPCase{
		{name: "exchange sets http cookie", method: http.MethodPost, path: "/candidate/access/exchange", origin: httpOrigin, body: `{"token":"invitation-secret"}`, callExchange: true, wantStatus: http.StatusCreated, wantCookie: "ia-candidate"},
		{name: "exchange sets https cookie", method: http.MethodPost, path: "/candidate/access/exchange", origin: httpsOrigin, body: `{"token":"invitation-secret"}`, secure: true, callExchange: true, wantStatus: http.StatusCreated, wantCookie: "__Host-ia-candidate"},
		{name: "exchange rejects assignment override", method: http.MethodPost, path: "/candidate/access/exchange", origin: httpOrigin, body: `{"token":"invitation-secret","assignmentId":"x"}`, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "unavailable invitation", method: http.MethodPost, path: "/candidate/access/exchange", origin: httpOrigin, body: `{"token":"invitation-secret"}`, callExchange: true, opErr: accessport.ErrInvitationUnavailable, wantStatus: http.StatusGone, wantCode: errs.IAInvitationUnavailable},
		{name: "uncertain exchange omits cookie", method: http.MethodPost, path: "/candidate/access/exchange", origin: httpOrigin, body: `{"token":"invitation-secret"}`, callExchange: true, opErr: &accessport.CommitUncertainError{}, wantStatus: http.StatusServiceUnavailable, wantCode: errs.IADependencyUnavailable},
		{name: "get exchange is not an exchange", method: http.MethodGet, path: "/candidate/access/exchange", origin: httpOrigin, wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
		{name: "head exchange is not an exchange", method: http.MethodHead, path: "/candidate/access/exchange", origin: httpOrigin, wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
		{name: "current session refresh", method: http.MethodGet, path: "/candidate/session", origin: httpOrigin, cookie: candidateCookie(false, "candidate-session-secret"), callSession: true, wantStatus: http.StatusOK},
		{name: "admin cookie excluded from candidate session", method: http.MethodGet, path: "/candidate/session", origin: httpOrigin, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "unbound candidate session", method: http.MethodGet, path: "/candidate/session", origin: httpOrigin, cookie: candidateCookie(false, "stale"), callSession: true, opErr: service.ErrCandidateSessionUnbound, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origin := httpOrigin
			if tt.secure {
				origin = httpsOrigin
			}
			handler, _, _, _, exchange, session, _, _ := newCandidateHandler(t, origin, tt.secure)
			if tt.callExchange {
				exchange.On("Execute", mock.Anything, "invitation-secret").Return(access, tt.opErr).Maybe()
			}
			if tt.callSession {
				session.On("Execute", mock.Anything).Return(access, tt.opErr).Maybe()
			}
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.RemoteAddr = "192.0.2.40:9"
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code)
			if tt.method == http.MethodHead {
				return
			}
			view := decodeEnvelope(t, rec)
			if tt.wantCode != "" {
				require.Equal(t, "error", view.Status)
				require.Equal(t, tt.wantCode, view.Error.Code)
				require.NotContains(t, rec.Header().Get("Set-Cookie"), "candidate-session-secret")
				return
			}
			require.Equal(t, "success", view.Status)
			require.NotContains(t, rec.Body.String(), "candidate-session-secret")
			require.NotContains(t, rec.Body.String(), testdata.PasswordHash)
			if tt.wantCookie != "" {
				found := false
				for _, cookie := range rec.Result().Cookies() {
					if cookie.Name != tt.wantCookie {
						continue
					}
					found = true
					require.Equal(t, "candidate-session-secret", cookie.Value)
					require.True(t, cookie.HttpOnly)
					require.Equal(t, "/", cookie.Path)
					require.Empty(t, cookie.Domain)
					require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
					require.Equal(t, tt.secure, cookie.Secure)
				}
				require.True(t, found)
			}
			if tt.callSession && tt.wantStatus == http.StatusOK {
				require.Contains(t, string(view.Data), csrfTok)
				var data map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(view.Data, &data))
				require.Contains(t, data, "csrfToken")
			}
		})
	}
}

// TestCandidateExchangeRateLimitBoundsAttempts refuses excess exchange attempts before the operation.
func TestCandidateExchangeRateLimitBoundsAttempts(t *testing.T) {
	handler, _, _, _, exchange, _, _, _ := newCandidateHandler(t, httpOrigin, false)
	body := `{"token":"invitation-secret"}`
	exchange.On("Execute", mock.Anything, "invitation-secret").Return((*dto.CandidateAccess)(nil), accessport.ErrInvitationUnavailable).Maybe()
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, request(http.MethodPost, "/candidate/access/exchange", httpOrigin, body, "", nil))
		require.Equal(t, http.StatusGone, rec.Code)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request(http.MethodPost, "/candidate/access/exchange", httpOrigin, body, "", nil))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	view := decodeEnvelope(t, rec)
	require.Equal(t, errs.IARateLimited, view.Error.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"))
}
