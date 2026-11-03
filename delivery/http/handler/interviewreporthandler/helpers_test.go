package interviewreporthandler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/interviewreporthandler"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
)

const (
	origin       = "https://hr.example.test"
	assignmentID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	readPath     = "/admin/interview-assignments/" + assignmentID + "/report"
	postPath     = "/admin/interview-assignments/" + assignmentID + "/report-generations"
)

// fixture holds independently asserted operation, session and CSRF mocks.
type fixture struct {
	h          interviewreporthandler.Handler
	read       *mocks.InterviewReportReadOperation
	regenerate *mocks.InterviewReportRegenerateOperation
	sessions   *mocks.CurrentAdministrativeSession
	csrf       *mocks.CSRFTokens
}

// newFixture wires generated mocks without permitting unexpected operation calls.
func newFixture(t *testing.T) fixture {
	t.Helper()
	f := fixture{read: mocks.NewInterviewReportReadOperation(t), regenerate: mocks.NewInterviewReportRegenerateOperation(t), sessions: mocks.NewCurrentAdministrativeSession(t), csrf: mocks.NewCSRFTokens(t)}
	f.h = interviewreporthandler.Handler{Read: f.read, Regenerate: f.regenerate, Sessions: f.sessions, CSRF: f.csrf, Origin: origin, SecureCookie: true}
	return f
}

// session binds a role to the trusted administrative cookie context.
func (f fixture) session(t *testing.T, role enum.Role, err error) {
	t.Helper()
	s := testdata.AdministrativeSessionRecord(t)
	a := testdata.AdministrativeAccountRecord(t)
	a.Role = role
	f.sessions.On("Resolve", mock.MatchedBy(func(ctx context.Context) bool {
		token, ok := service.AdministrativeCredentialFrom(ctx)
		return ok && token == "session-secret"
	})).Return(s, a, err).Once()
}

// request supplies a secure administrative cookie, Origin and mutation headers.
func request(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Origin", origin)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "report-1")
	r.Header.Set("X-CSRF-Token", "csrf-secret")
	r.AddCookie(&http.Cookie{Name: "__Host-ia-admin", Value: "session-secret"})
	return r
}

// allowCSRF accepts the session-bound mutation token exactly once.
func (f fixture) allowCSRF(t *testing.T, err error) {
	t.Helper()
	f.csrf.On("Verify", service.CredentialAdministrative, testdata.AdministrativeSessionRecord(t).ID, "csrf-secret").Return(err).Once()
}

// generation returns a fresh detached pending generation at a non-UTC nanosecond instant.
func generation() *dto.ReportGeneration {
	return &dto.ReportGeneration{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Number: 2, State: "pending", CreatedAt: time.Date(2026, 11, 3, 10, 0, 0, 123456789, time.FixedZone("WIB", 7*3600))}
}
