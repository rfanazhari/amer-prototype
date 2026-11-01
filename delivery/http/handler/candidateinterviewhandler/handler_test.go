package candidateinterviewhandler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/candidateinterviewhandler"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	httpOrigin  = "http://localhost:8080"
	csrfToken   = "signed-csrf-material"
	sessionTok  = "candidate-session-secret"
	noticeV     = "notice-261102"
	consentID   = "11111111-1111-4111-8111-111111111111"
	controller  = "tab-1"
	jsonContent = "application/json"
)

// envelopeView is the decoded public JSON envelope used by transport assertions.
type envelopeView struct {
	Status string         `json:"status"`
	Data   map[string]any `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// graph holds the mocked collaborators of one handler scenario.
type graph struct {
	get     *mocks.GetInterviewOperation
	accept  *mocks.AcceptNoticeOperation
	current *mocks.CurrentCandidateSession
	csrf    *mocks.CSRFTokens
}

// newGraph builds strict mocks and the handler wired to them.
func newGraph(t *testing.T) (candidateinterviewhandler.Handler, graph) {
	t.Helper()
	g := graph{
		get:     mocks.NewGetInterviewOperation(t),
		accept:  mocks.NewAcceptNoticeOperation(t),
		current: mocks.NewCurrentCandidateSession(t),
		csrf:    mocks.NewCSRFTokens(t),
	}
	return candidateinterviewhandler.Handler{
		Get: g.get, Accept: g.accept, Current: g.current, CSRF: g.csrf, Origin: httpOrigin,
	}, g
}

// cookie returns the candidate session cookie for the plain-HTTP configuration.
func cookie() *http.Cookie { return &http.Cookie{Name: "ia-candidate", Value: sessionTok} }

// do sends one request through the handler and decodes the envelope.
func do(t *testing.T, h http.Handler, method, path, body string, mutate func(*http.Request)) (*httptest.ResponseRecorder, envelopeView) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "localhost:8080"
	if method == http.MethodPost {
		req.Header.Set("Content-Type", jsonContent)
		req.Header.Set("Origin", httpOrigin)
	}
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var view envelopeView
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view), rec.Body.String())
	return rec, view
}

// withCookie attaches the candidate cookie.
func withCookie(r *http.Request) { r.AddCookie(cookie()) }

// withCSRF attaches the candidate cookie and the CSRF header.
func withCSRF(r *http.Request) {
	r.AddCookie(cookie())
	r.Header.Set("X-CSRF-Token", csrfToken)
}

// expectCSRFAccepted makes the session resolve and the CSRF value verify.
func expectCSRFAccepted(g graph, t *testing.T) {
	t.Helper()
	session := testdata.CandidateSessionRecord(t)
	g.current.On("Resolve", mock.Anything).Return(session, nil)
	g.csrf.On("Verify", service.CredentialCandidate, session.ID, csrfToken).Return(nil)
}

// at returns the fixed acceptance instant of the fixtures.
func at() time.Time { return time.Date(2026, 11, 2, 8, 5, 0, 123_000_000, time.UTC) }

// errorCode asserts an error envelope and returns its code.
func errorCode(t *testing.T, rec *httptest.ResponseRecorder, view envelopeView, status int) string {
	t.Helper()
	require.Equal(t, status, rec.Code)
	require.Equal(t, "error", view.Status)
	require.NotNil(t, view.Error)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	return view.Error.Code
}

// getCase names one GET /candidate/interview transport outcome.
type getCase struct {
	name       string
	path       string
	method     string
	mutate     func(*http.Request)
	view       *dto.InterviewView
	opErr      error
	callOp     bool
	wantStatus int
	wantCode   string
}

// TestGetInterviewTransport covers origin, cookie, operation errors and the success envelope of the interview read.
func TestGetInterviewTransport(t *testing.T) {
	empty := &dto.InterviewView{Actions: []string{}}
	tests := []getCase{
		{name: "success", path: "/candidate/interview", method: http.MethodGet, mutate: withCookie, view: empty, callOp: true, wantStatus: http.StatusOK},
		{name: "wrong method", path: "/candidate/interview", method: http.MethodPost, mutate: withCookie, wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
		{name: "foreign origin", path: "/candidate/interview", method: http.MethodGet, mutate: func(r *http.Request) { withCookie(r); r.Header.Set("Origin", "https://evil.example") }, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "missing cookie", path: "/candidate/interview", method: http.MethodGet, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "administrative cookie is not a candidate", path: "/candidate/interview", method: http.MethodGet, mutate: func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "ia-admin", Value: "admin-secret"}) }, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "session ended", path: "/candidate/interview", method: http.MethodGet, mutate: withCookie, callOp: true, opErr: service.ErrCandidateSessionUnbound, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "technical failure is safe", path: "/candidate/interview", method: http.MethodGet, mutate: withCookie, callOp: true, opErr: errors.New("pq: password=hunter2"), wantStatus: http.StatusInternalServerError, wantCode: errs.InternalError},
		{name: "unknown candidate path", path: "/candidate/unknown", method: http.MethodGet, mutate: withCookie, wantStatus: http.StatusNotFound, wantCode: errs.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, g := newGraph(t)
			if tt.callOp {
				g.get.On("Execute", mock.Anything, candidateinterviewquery.GetInterviewInput{}).Return(tt.view, tt.opErr)
			}
			rec, view := do(t, h, tt.method, tt.path, "", tt.mutate)
			if tt.wantCode != "" {
				require.Equal(t, tt.wantCode, errorCode(t, rec, view, tt.wantStatus))
				require.NotContains(t, rec.Body.String(), "hunter2")
				return
			}
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, "success", view.Status)
			require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			require.Contains(t, rec.Header().Get("Content-Type"), jsonContent)
		})
	}
}

// TestGetInterviewForwardsTabAndEncodesAllowList verifies the controller header reaches the query and the body is the allow-list.
func TestGetInterviewForwardsTabAndEncodesAllowList(t *testing.T) {
	deadline := time.Date(2026, 11, 2, 8, 20, 0, 0, time.UTC)
	full := &dto.InterviewView{
		Notice:    &dto.NoticeView{Version: noticeV, Statement: "statement"},
		Consent:   &dto.ConsentView{ID: consentID, NoticeVersion: noticeV, AcceptedAt: at()},
		Execution: &dto.InterviewExecutionView{ID: "e1", State: "RUNNING", Version: 2, HardDeadline: &deadline, IsCurrentController: true, ControllerVersion: 3},
		Actions:   []string{"cancel"},
		Progress:  &dto.ProgressView{Phase: "screening", Connection: "connected"},
	}
	h, g := newGraph(t)
	g.get.On("Execute", mock.Anything, candidateinterviewquery.GetInterviewInput{ControllerID: controller}).Return(full, nil)

	rec, view := do(t, h, http.MethodGet, "/candidate/interview", "", func(r *http.Request) {
		withCookie(r)
		r.Header.Set("X-Interview-Controller", controller)
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.ElementsMatch(t, []string{"notice", "consent", "execution", "actions", "progress"}, keys(view.Data))
	require.Equal(t, map[string]any{"version": noticeV, "statement": "statement"}, view.Data["notice"])
	require.Equal(t, map[string]any{"id": consentID, "noticeVersion": noticeV, "acceptedAt": "2026-11-02T08:05:00.123Z"}, view.Data["consent"])
	require.Equal(t, map[string]any{
		"id": "e1", "state": "RUNNING", "version": float64(2), "hardDeadline": "2026-11-02T08:20:00.000Z",
		"isCurrentController": true, "controllerVersion": float64(3),
	}, view.Data["execution"])
	require.Equal(t, []any{"cancel"}, view.Data["actions"])
	require.Equal(t, map[string]any{"phase": "screening", "connection": "connected"}, view.Data["progress"])
	lower := strings.ToLower(rec.Body.String())
	for _, forbidden := range []string{"salary", "report", "controllerid", "transcript", controller} {
		require.NotContains(t, lower, forbidden)
	}
}

// TestGetInterviewRendersNullsAndEmptyActions verifies absent members are JSON null and actions is an array, never null.
func TestGetInterviewRendersNullsAndEmptyActions(t *testing.T) {
	h, g := newGraph(t)
	g.get.On("Execute", mock.Anything, candidateinterviewquery.GetInterviewInput{}).Return(&dto.InterviewView{}, nil)

	rec, view := do(t, h, http.MethodGet, "/candidate/interview", "", withCookie)

	require.Equal(t, http.StatusOK, rec.Code)
	for _, member := range []string{"notice", "consent", "execution", "progress"} {
		value, present := view.Data[member]
		require.True(t, present, member)
		require.Nil(t, value, member)
	}
	require.Equal(t, []any{}, view.Data["actions"])
}

// TestGetInterviewAcceptsSameOriginGetWithoutOrigin verifies the browser same-origin GET that omits Origin.
func TestGetInterviewAcceptsSameOriginGetWithoutOrigin(t *testing.T) {
	h, g := newGraph(t)
	g.get.On("Execute", mock.Anything, candidateinterviewquery.GetInterviewInput{}).Return(&dto.InterviewView{Actions: []string{}}, nil)

	rec, _ := do(t, h, http.MethodGet, "/candidate/interview", "", withCookie)

	require.Equal(t, http.StatusOK, rec.Code)
}

// postCase names one POST /candidate/consents transport outcome.
type postCase struct {
	name       string
	body       string
	mutate     func(*http.Request)
	session    string
	input      *candidateinterviewusecase.AcceptNoticeInput
	result     *candidateinterviewusecase.AcceptNoticeResult
	opErr      error
	wantStatus int
	wantCode   string
}

// TestAcceptNoticeTransport covers strict JSON, precedence, CSRF, operation errors and the 201/200 replay contract.
func TestAcceptNoticeTransport(t *testing.T) {
	good := `{"noticeVersion":"` + noticeV + `","accepted":true}`
	input := &candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: noticeV, Accepted: true}
	created := &candidateinterviewusecase.AcceptNoticeResult{Consent: dto.ConsentView{ID: consentID, NoticeVersion: noticeV, AcceptedAt: at()}, Created: true}
	replay := &candidateinterviewusecase.AcceptNoticeResult{Consent: created.Consent}
	tests := []postCase{
		{name: "first acceptance is 201", body: good, mutate: withCSRF, session: "ok", input: input, result: created, wantStatus: http.StatusCreated},
		{name: "retry is 200", body: good, mutate: withCSRF, session: "ok", input: input, result: replay, wantStatus: http.StatusOK},
		{name: "unknown field", body: `{"noticeVersion":"` + noticeV + `","accepted":true,"id":"x"}`, mutate: withCSRF, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "system-owned timestamp is rejected", body: `{"noticeVersion":"` + noticeV + `","accepted":true,"acceptedAt":"2026-11-02T08:05:00.000Z"}`, mutate: withCSRF, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "malformed json", body: `{"noticeVersion":`, mutate: withCSRF, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "trailing document", body: good + `{}`, mutate: withCSRF, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "empty body", body: ``, mutate: withCSRF, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "wrong content type", body: good, mutate: func(r *http.Request) { withCSRF(r); r.Header.Set("Content-Type", "text/plain") }, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "foreign origin", body: good, mutate: func(r *http.Request) { withCSRF(r); r.Header.Set("Origin", "https://evil.example") }, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "malformed body wins over missing cookie", body: `{"x":1}`, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "missing cookie", body: good, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "session already ended", body: good, mutate: withCSRF, session: "ended", wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "accepted false is invalid input", body: `{"noticeVersion":"` + noticeV + `","accepted":false}`, mutate: withCSRF, session: "ok", input: &candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: noticeV}, opErr: candidateinterview.ErrInvalidConsent, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "malformed notice version is invalid input", body: `{"noticeVersion":" bad","accepted":true}`, mutate: withCSRF, session: "ok", input: &candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: " bad", Accepted: true}, opErr: candidateinterview.ErrInvalidNoticeVersion, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "stale notice version is invalid input", body: good, mutate: withCSRF, session: "ok", input: input, opErr: candidateinterview.ErrNoticeNotCurrent, wantStatus: http.StatusBadRequest, wantCode: errs.InvalidInput},
		{name: "empty catalog", body: good, mutate: withCSRF, session: "ok", input: input, opErr: service.ErrInterviewDependencyUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: errs.CINNoticeUnavailable},
		{name: "revoked assignment", body: good, mutate: withCSRF, session: "ok", input: input, opErr: service.ErrCandidateSessionUnbound, wantStatus: http.StatusUnauthorized, wantCode: errs.IAUnauthenticated},
		{name: "technical failure is safe", body: good, mutate: withCSRF, session: "ok", input: input, opErr: errors.New("pq: password=hunter2"), wantStatus: http.StatusInternalServerError, wantCode: errs.InternalError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, g := newGraph(t)
			record := testdata.CandidateSessionRecord(t)
			switch tt.session {
			case "ok":
				g.current.On("Resolve", mock.Anything).Return(record, nil)
				g.csrf.On("Verify", service.CredentialCandidate, record.ID, csrfToken).Return(nil)
			case "ended":
				g.current.On("Resolve", mock.Anything).Return(accessport.CandidateSessionRecord{}, service.ErrCandidateSessionUnbound)
			}
			if tt.input != nil {
				g.accept.On("Execute", mock.Anything, *tt.input).Return(tt.result, tt.opErr)
			}
			rec, view := do(t, h, http.MethodPost, "/candidate/consents", tt.body, tt.mutate)
			if tt.wantCode != "" {
				require.Equal(t, tt.wantCode, errorCode(t, rec, view, tt.wantStatus))
				require.NotContains(t, rec.Body.String(), "hunter2")
				return
			}
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, "success", view.Status)
			require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			require.Equal(t, map[string]any{"id": consentID, "noticeVersion": noticeV, "acceptedAt": "2026-11-02T08:05:00.123Z"}, view.Data)
		})
	}
}

// TestAcceptNoticeRejectsBadCSRFBeforeAnyOperation verifies a missing or mismatched CSRF value is 403 CIN_FORBIDDEN and writes nothing.
func TestAcceptNoticeRejectsBadCSRFBeforeAnyOperation(t *testing.T) {
	body := `{"noticeVersion":"` + noticeV + `","accepted":true}`
	tests := []struct {
		name   string
		mutate func(*http.Request)
		verify error
	}{
		{name: "missing header", mutate: withCookie},
		{name: "mismatched value", mutate: withCSRF, verify: service.ErrCSRFMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, g := newGraph(t)
			session := testdata.CandidateSessionRecord(t)
			g.current.On("Resolve", mock.Anything).Return(session, nil)
			if tt.verify != nil {
				g.csrf.On("Verify", service.CredentialCandidate, session.ID, csrfToken).Return(tt.verify)
			}

			rec, view := do(t, h, http.MethodPost, "/candidate/consents", body, tt.mutate)

			require.Equal(t, errs.CINForbidden, errorCode(t, rec, view, http.StatusForbidden))
			g.accept.AssertNotCalled(t, "Execute", mock.Anything, mock.Anything)
		})
	}
}

// TestAcceptNoticeRejectsGet verifies the consent collection only accepts POST.
func TestAcceptNoticeRejectsGet(t *testing.T) {
	h, _ := newGraph(t)

	rec, view := do(t, h, http.MethodGet, "/candidate/consents", "", withCookie)

	require.Equal(t, errs.MethodNotAllowed, errorCode(t, rec, view, http.StatusMethodNotAllowed))
}

// TestErrorMessagesAreLocalized verifies CIN_ messages follow Accept-Language in English and Indonesian.
func TestErrorMessagesAreLocalized(t *testing.T) {
	tests := []struct {
		name, language string
	}{
		{name: "english", language: "en"},
		{name: "indonesian", language: "id"},
	}
	messages := map[string]string{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, g := newGraph(t)
			expectCSRFAccepted(g, t)
			g.accept.On("Execute", mock.Anything, mock.Anything).Return(nil, service.ErrInterviewDependencyUnavailable)

			rec, view := do(t, h, http.MethodPost, "/candidate/consents", `{"noticeVersion":"`+noticeV+`","accepted":true}`, func(r *http.Request) {
				withCSRF(r)
				r.Header.Set("Accept-Language", tt.language)
			})

			require.Equal(t, errs.CINNoticeUnavailable, errorCode(t, rec, view, http.StatusServiceUnavailable))
			require.NotEmpty(t, view.Error.Message)
			messages[tt.language] = view.Error.Message
		})
	}
	require.NotEqual(t, messages["en"], messages["id"])
}

// keys returns the member names of a decoded JSON object.
func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
