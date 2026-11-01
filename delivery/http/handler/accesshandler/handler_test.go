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
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	httpOrigin  = "http://localhost:8080"
	httpsOrigin = "https://app.example.test"
	sessionTok  = "plain-session-secret"
	csrfTok     = "signed-csrf-material"
)

// envelopeView decodes the public JSON envelope used by administrative HTTP assertions.
type envelopeView struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// loginHTTPCase names one login transport outcome against a mocked LoginOperation.
type loginHTTPCase struct {
	name        string
	origin      string
	contentType string
	body        string
	secure      bool
	callLogin   bool
	loginErr    error
	wantStatus  int
	wantCode    string
	wantCookie  string
	wantSecure  bool
}

// currentHTTPCase names one current-session transport outcome.
type currentHTTPCase struct {
	name       string
	origin     string
	cookie     *http.Cookie
	wantStatus int
	wantCode   string
}

// protectedHTTPCase names one CSRF-protected administrative mutation outcome.
type protectedHTTPCase struct {
	name        string
	method      string
	path        string
	body        string
	csrf        string
	cookie      *http.Cookie
	roleDenied  bool
	logoutErr   error
	wantStatus  int
	wantCode    string
	clearCookie bool
}

// newHandler wires generated operation mocks for one administrative HTTP scenario.
func newHandler(t *testing.T, origin string, secure bool) (accesshandler.Handler, *mocks.LoginOperation, *mocks.CurrentSessionOperation, *mocks.ProvisionHROperation, *mocks.LogoutOperation, *mocks.CSRFTokens, *mocks.CurrentAdministrativeSession) {
	t.Helper()
	login := mocks.NewLoginOperation(t)
	current := mocks.NewCurrentSessionOperation(t)
	provision := mocks.NewProvisionHROperation(t)
	logout := mocks.NewLogoutOperation(t)
	csrf := mocks.NewCSRFTokens(t)
	sessions := mocks.NewCurrentAdministrativeSession(t)
	return accesshandler.Handler{
		Login:        login,
		Current:      current,
		Provision:    provision,
		Logout:       logout,
		CSRF:         csrf,
		Sessions:     sessions,
		Origin:       origin,
		SecureCookie: secure,
		Limiter:      &accesshandler.LoginLimiter{},
	}, login, current, provision, logout, csrf, sessions
}

// request builds an administrative JSON request with origin, cookie and CSRF headers.
func request(method, path, origin, body, csrf string, cookie *http.Cookie) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:1234"
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return req
}

// decodeEnvelope reads the standard envelope and requires Cache-Control no-store.
func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelopeView {
	t.Helper()
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var view envelopeView
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
	return view
}

// adminCookie returns the configured administrative session cookie name for an origin.
func adminCookie(secure bool, value string) *http.Cookie {
	name := "ia-admin"
	if secure {
		name = "__Host-ia-admin"
	}
	return &http.Cookie{Name: name, Value: value}
}

// TestLoginEstablishesCookieAfterCommit covers origin, cookies and no-store login transport.
func TestLoginEstablishesCookieAfterCommit(t *testing.T) {
	account := testdata.AdministrativeAccountRecord(t)
	session := testdata.AdministrativeSessionRecord(t)
	result := &dto.AdministrativeLogin{
		Account:      dto.FromAdministrativeAccount(account),
		Session:      dto.FromAdministrativeSession(session),
		SessionToken: sessionTok,
	}
	tests := []loginHTTPCase{
		{name: "http localhost cookie", origin: httpOrigin, contentType: "application/json", body: `{"email":"Sinta@Example.com","password":"correct-horse"}`, callLogin: true, wantStatus: http.StatusCreated, wantCookie: "ia-admin", wantSecure: false},
		{name: "https host cookie", origin: httpsOrigin, contentType: "application/json", body: `{"email":"Sinta@Example.com","password":"correct-horse"}`, secure: true, callLogin: true, wantStatus: http.StatusCreated, wantCookie: "__Host-ia-admin", wantSecure: true},
		{name: "origin mismatch", origin: "https://evil.example", contentType: "application/json", body: `{"email":"sinta@example.com","password":"correct-horse"}`, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "missing origin", contentType: "application/json", body: `{"email":"sinta@example.com","password":"correct-horse"}`, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "wrong content type", origin: httpOrigin, contentType: "text/plain", body: `{"email":"sinta@example.com","password":"correct-horse"}`, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "unknown field", origin: httpOrigin, contentType: "application/json", body: `{"email":"sinta@example.com","password":"correct-horse","role":"ADMIN"}`, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "wrong password", origin: httpOrigin, contentType: "application/json", body: `{"email":"sinta@example.com","password":"wrong-horse"}`, callLogin: true, loginErr: service.ErrAuthenticationFailed, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "uncertain commit", origin: httpOrigin, contentType: "application/json", body: `{"email":"sinta@example.com","password":"correct-horse"}`, callLogin: true, loginErr: &accessport.CommitUncertainError{}, wantStatus: http.StatusServiceUnavailable, wantCode: errs.IADependencyUnavailable},
		{name: "wrong method", origin: httpOrigin, contentType: "application/json", body: `{"email":"sinta@example.com","password":"correct-horse"}`, wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, login, _, _, _, _, _ := newHandler(t, func() string {
				if tt.secure {
					return httpsOrigin
				}
				return httpOrigin
			}(), tt.secure)
			if tt.callLogin {
				login.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return(result, tt.loginErr).Once()
			}
			method := http.MethodPost
			if tt.name == "wrong method" {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, "/admin/sessions", strings.NewReader(tt.body))
			req.RemoteAddr = "192.0.2.10:9"
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, tt.wantStatus, rec.Code)
			view := decodeEnvelope(t, rec)
			if tt.wantCode != "" {
				require.Equal(t, "error", view.Status)
				require.Equal(t, tt.wantCode, view.Error.Code)
				require.NotContains(t, rec.Header().Get("Set-Cookie"), sessionTok)
				return
			}
			require.Equal(t, "success", view.Status)
			require.NotContains(t, rec.Body.String(), sessionTok)
			require.NotContains(t, rec.Body.String(), testdata.PasswordHash)
			cookies := rec.Result().Cookies()
			require.NotEmpty(t, cookies)
			found := false
			for _, cookie := range cookies {
				if cookie.Name != tt.wantCookie {
					continue
				}
				found = true
				require.Equal(t, sessionTok, cookie.Value)
				require.True(t, cookie.HttpOnly)
				require.Equal(t, "/", cookie.Path)
				require.Empty(t, cookie.Domain)
				require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
				require.Equal(t, tt.wantSecure, cookie.Secure)
			}
			require.True(t, found)
		})
	}
}

// TestLoginRateLimitBoundsAttempts refuses excess login attempts before the login operation.
func TestLoginRateLimitBoundsAttempts(t *testing.T) {
	handler, login, _, _, _, _, _ := newHandler(t, httpOrigin, false)
	body := `{"email":"sinta@example.com","password":"correct-horse"}`
	login.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return((*dto.AdministrativeLogin)(nil), service.ErrAuthenticationFailed)
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := request(http.MethodPost, "/admin/sessions", httpOrigin, body, "", nil)
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request(http.MethodPost, "/admin/sessions", httpOrigin, body, "", nil))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	view := decodeEnvelope(t, rec)
	require.Equal(t, errs.IARateLimited, view.Error.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"))
}

// TestCurrentSessionAndCandidateCookie covers current session CSRF material and candidate-cookie exclusion.
func TestCurrentSessionAndCandidateCookie(t *testing.T) {
	account := testdata.AdministrativeAccountRecord(t)
	account.Role = enum.HR
	session := testdata.AdministrativeSessionRecord(t)
	view := &dto.AdministrativeCurrentSession{
		Account:   dto.FromAdministrativeAccount(account),
		Session:   dto.FromAdministrativeSession(session),
		CSRFToken: csrfTok,
	}
	tests := []currentHTTPCase{
		{name: "current session", origin: httpOrigin, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusOK},
		{name: "missing cookie", origin: httpOrigin, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "candidate cookie excluded", origin: httpOrigin, cookie: &http.Cookie{Name: "ia-candidate", Value: sessionTok}, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "origin mismatch", origin: "https://evil.example", cookie: adminCookie(false, sessionTok), wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, current, _, _, _, sessions := newHandler(t, httpOrigin, false)
			if tt.wantStatus == http.StatusOK {
				sessions.On("Resolve", mock.Anything).Return(testdata.AdministrativeSessionRecord(t), testdata.AdministrativeAccountRecord(t), nil).Maybe()
				current.On("Execute", mock.Anything).Return(view, nil).Once()
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request(http.MethodGet, "/admin/session", tt.origin, "", "", tt.cookie))
			require.Equal(t, tt.wantStatus, rec.Code)
			got := decodeEnvelope(t, rec)
			if tt.wantCode != "" {
				require.Equal(t, tt.wantCode, got.Error.Code)
				return
			}
			require.Equal(t, "success", got.Status)
			require.Contains(t, string(got.Data), csrfTok)
			require.NotContains(t, rec.Header().Get("Set-Cookie"), sessionTok)
		})
	}
}

// TestProtectedMutationsEnforceCSRFAndRole covers provision, logout, CSRF and wrong-role denial.
func TestProtectedMutationsEnforceCSRFAndRole(t *testing.T) {
	account := testdata.AdministrativeAccountRecord(t)
	account.Role = enum.HR
	provisioned := dto.FromAdministrativeAccount(account)
	tests := []protectedHTTPCase{
		{name: "provision hr", method: http.MethodPost, path: "/admin/hr-accounts", body: `{"name":" Sinta Wijaya ","email":"Sinta@Example.com","password":"correct-horse"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusCreated},
		{name: "missing csrf", method: http.MethodPost, path: "/admin/hr-accounts", body: `{"name":"Sinta Wijaya","email":"sinta@example.com","password":"correct-horse"}`, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusForbidden, wantCode: errs.IAForbidden},
		{name: "hr forbidden", method: http.MethodPost, path: "/admin/hr-accounts", body: `{"name":"Sinta Wijaya","email":"sinta@example.com","password":"correct-horse"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), roleDenied: true, wantStatus: http.StatusForbidden, wantCode: errs.IAForbidden},
		{name: "duplicate email", method: http.MethodPost, path: "/admin/hr-accounts", body: `{"name":"Sinta Wijaya","email":"sinta@example.com","password":"correct-horse"}`, csrf: csrfTok, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusConflict, wantCode: errs.IAAccountConflict},
		{name: "logout success", method: http.MethodDelete, path: "/admin/sessions/current", csrf: csrfTok, cookie: adminCookie(false, sessionTok), wantStatus: http.StatusOK, clearCookie: true},
		{name: "logout uncertain still clears cookie", method: http.MethodDelete, path: "/admin/sessions/current", csrf: csrfTok, cookie: adminCookie(false, sessionTok), logoutErr: &accessport.CommitUncertainError{}, wantStatus: http.StatusServiceUnavailable, wantCode: errs.IADependencyUnavailable, clearCookie: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, _, provision, logout, csrf, sessions := newHandler(t, httpOrigin, false)
			session := testdata.AdministrativeSessionRecord(t)
			accountRec := testdata.AdministrativeAccountRecord(t)
			if tt.cookie != nil && tt.cookie.Name == "ia-admin" {
				sessions.On("Resolve", mock.Anything).Return(session, accountRec, nil)
			}
			if tt.csrf != "" {
				csrf.On("Verify", service.CredentialAdministrative, session.ID, tt.csrf).Return(nil)
			}
			switch {
			case tt.path == "/admin/hr-accounts" && tt.wantStatus == http.StatusCreated:
				provision.On("Execute", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&provisioned, nil).Once()
			case tt.roleDenied:
				provision.On("Execute", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*dto.AdministrativeAccount)(nil), service.ErrForbidden).Once()
			case tt.wantCode == errs.IAAccountConflict:
				provision.On("Execute", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*dto.AdministrativeAccount)(nil), accessport.AccountConflictError{}).Once()
			case tt.path == "/admin/sessions/current" && tt.csrf != "":
				logout.On("Execute", mock.Anything).Return(tt.logoutErr).Once()
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request(tt.method, tt.path, httpOrigin, tt.body, tt.csrf, tt.cookie))
			require.Equal(t, tt.wantStatus, rec.Code)
			view := decodeEnvelope(t, rec)
			if tt.wantCode != "" {
				require.Equal(t, tt.wantCode, view.Error.Code)
			} else {
				require.Equal(t, "success", view.Status)
			}
			if tt.clearCookie {
				cookies := rec.Result().Cookies()
				require.NotEmpty(t, cookies)
				require.True(t, cookies[0].MaxAge < 0 || cookies[0].Value == "")
			}
		})
	}
}
