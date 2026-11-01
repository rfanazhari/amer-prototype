package candidateinterviewhandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const executionID = "22222222-2222-4222-8222-222222222222"

// executionCase describes a candidate execution transport scenario and its expected operation boundary.
type executionCase struct {
	name    string
	route   string
	method  string
	body    string
	mutate  func(*http.Request)
	session string
	call    bool
	created bool
	opErr   error
	status  int
	code    string
}

// executionHeaders sends one tab's candidate credentials, CSRF and repeatable start intent.
func executionHeaders(r *http.Request) {
	withCSRF(r)
	r.Header.Set("X-Interview-Controller", controller)
	r.Header.Set("Idempotency-Key", "start-1")
}

// TestExecutionTransport covers start replay, poll, takeover, cancellation, safe errors and validation precedence.
func TestExecutionTransport(t *testing.T) {
	startBody := `{"consentId":"` + consentID + `"}`
	takeoverBody := `{"controllerId":"tab-1","expectedVersion":1}`
	cancelBody := `{"expectedVersion":1}`
	tests := []executionCase{
		{name: "not live", route: "cancel", body: cancelBody, session: "ok", call: true, opErr: service.ErrInterviewNotLive, status: 409, code: errs.CINNotLive},
		{name: "ambiguous start", route: "start", body: startBody, session: "ok", call: true, opErr: service.ErrInterviewStartUncertain, status: 503, code: errs.CINStartUncertain},
		{name: "start created", route: "start", body: startBody, session: "ok", call: true, created: true, status: 201},
		{name: "start replay", route: "start", body: startBody, session: "ok", call: true, status: 200},
		{name: "poll", route: "poll", session: "ok", call: true, status: 200},
		{name: "takeover", route: "takeover", body: takeoverBody, session: "ok", call: true, status: 200},
		{name: "cancel", route: "cancel", body: cancelBody, session: "ok", call: true, status: 200},
		{name: "foreign execution", route: "poll", session: "ok", call: true, opErr: service.ErrInterviewNotFound, status: 404, code: errs.NotFound},
		{name: "foreign takeover", route: "takeover", body: takeoverBody, session: "ok", call: true, opErr: service.ErrInterviewNotFound, status: 404, code: errs.NotFound},
		{name: "foreign cancel", route: "cancel", body: cancelBody, session: "ok", call: true, opErr: service.ErrInterviewNotFound, status: 404, code: errs.NotFound},
		{name: "consent required", route: "start", body: startBody, session: "ok", call: true, opErr: candidateinterviewusecase.ErrConsentRequired, status: 409, code: errs.CINConsentRequired},
		{name: "coverage", route: "start", body: startBody, session: "ok", call: true, opErr: service.ErrInterviewCoverage, status: 403, code: errs.CINSessionCoverage},
		{name: "intent conflict", route: "start", body: startBody, session: "ok", call: true, opErr: service.ErrInterviewConflict, status: 409, code: errs.CINIdempotencyConflict},
		{name: "controller conflict", route: "takeover", body: takeoverBody, session: "ok", call: true, opErr: service.ErrInterviewConflict, status: 409, code: errs.CINControllerConflict},
		{name: "cancel stale", route: "cancel", body: cancelBody, session: "ok", call: true, opErr: service.ErrInterviewConflict, status: 409, code: errs.CINControllerConflict},
		{name: "not controller", route: "cancel", body: cancelBody, session: "ok", call: true, opErr: candidateinterviewusecase.ErrNotController, status: 403, code: errs.CINNotController},
		{name: "terminal", route: "cancel", body: cancelBody, session: "ok", call: true, opErr: candidateinterview.ErrExecutionTerminal, status: 409, code: errs.CINTerminal},
		{name: "terminal takeover", route: "takeover", body: takeoverBody, session: "ok", call: true, opErr: candidateinterview.ErrExecutionTerminal, status: 409, code: errs.CINTerminal},
		{name: "unbound start", route: "start", body: startBody, session: "ok", call: true, opErr: service.ErrInterviewDependencyUnavailable, status: 503, code: errs.CINDependencyUnavailable},
		{name: "unbound cancel", route: "cancel", body: cancelBody, session: "ok", call: true, opErr: service.ErrInterviewDependencyUnavailable, status: 503, code: errs.CINDependencyUnavailable},
		{name: "technical safe", route: "poll", session: "ok", call: true, opErr: errors.New("password=hunter2"), status: 500, code: errs.InternalError},
		{name: "method before origin", route: "start", method: "DELETE", status: 405, code: errs.MethodNotAllowed},
		{name: "poll wrong method", route: "poll", method: "POST", status: 405, code: errs.MethodNotAllowed},
		{name: "takeover wrong method", route: "takeover", method: "GET", status: 405, code: errs.MethodNotAllowed},
		{name: "cancel wrong method", route: "cancel", method: "GET", status: 405, code: errs.MethodNotAllowed},
		{name: "bad key before session", route: "start", body: startBody, mutate: func(r *http.Request) { r.Header.Set("Idempotency-Key", " bad") }, status: 400, code: errs.InvalidInput},
		{name: "missing key before session", route: "start", body: startBody, mutate: func(r *http.Request) { r.Header.Del("Idempotency-Key") }, status: 400, code: errs.InvalidInput},
		{name: "bad consent before session", route: "start", body: `{"consentId":"invalid"}`, status: 400, code: errs.InvalidInput},
		{name: "unknown field", route: "start", body: `{"consentId":"` + consentID + `","id":"x"}`, status: 400, code: errs.InvalidInput},
		{name: "duplicate field", route: "cancel", body: `{"expectedVersion":1,"expectedVersion":2}`, status: 400, code: errs.InvalidInput},
		{name: "null version", route: "cancel", body: `{"expectedVersion":null}`, status: 400, code: errs.InvalidInput},
		{name: "missing version", route: "cancel", body: `{}`, status: 400, code: errs.InvalidInput},
		{name: "negative version", route: "takeover", body: `{"controllerId":"tab-1","expectedVersion":-1}`, status: 400, code: errs.InvalidInput},
		{name: "controller mismatch", route: "takeover", body: `{"controllerId":"tab-2","expectedVersion":1}`, status: 400, code: errs.InvalidInput},
		{name: "invalid tab", route: "cancel", body: cancelBody, mutate: func(r *http.Request) { r.Header.Set("X-Interview-Controller", " bad") }, status: 400, code: errs.InvalidInput},
		{name: "trailing json", route: "cancel", body: cancelBody + `{}`, status: 400, code: errs.InvalidInput},
		{name: "content type before session", route: "cancel", body: cancelBody, mutate: func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, status: 400, code: errs.InvalidInput},
		{name: "origin before session", route: "poll", mutate: func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, status: 400, code: errs.InvalidInput},
		{name: "missing session", route: "cancel", body: cancelBody, mutate: func(r *http.Request) { r.Header.Del("Cookie") }, status: 401, code: errs.IAUnauthenticated},
		{name: "ended before csrf", route: "cancel", body: cancelBody, session: "ended", status: 401, code: errs.IAUnauthenticated},
		{name: "missing csrf", route: "cancel", body: cancelBody, session: "no-csrf", mutate: func(r *http.Request) { r.Header.Del("X-CSRF-Token") }, status: 403, code: errs.CINForbidden},
		{name: "bad csrf", route: "takeover", body: takeoverBody, session: "bad-csrf", status: 403, code: errs.CINForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, g := newGraph(t)
			start, poll, takeover, cancel := mocks.NewStartInterviewOperation(t), mocks.NewGetExecutionOperation(t), mocks.NewTakeoverOperation(t), mocks.NewCancelInterviewOperation(t)
			h.Start, h.GetExecution, h.Takeover, h.Cancel = start, poll, takeover, cancel
			facts := dto.InterviewExecutionView{ID: executionID, State: "CREATED", Version: 1, IsCurrentController: true, ControllerVersion: 1}
			if tt.session != "" && tt.route != "poll" {
				session := testdata.CandidateSessionRecord(t)
				if tt.session == "ended" {
					g.current.On("Resolve", mock.Anything).Return(session, service.ErrCandidateSessionUnbound)
				} else {
					g.current.On("Resolve", mock.Anything).Return(session, nil)
					if tt.session != "no-csrf" {
						var err error
						if tt.session == "bad-csrf" {
							err = service.ErrCSRFMismatch
						}
						g.csrf.On("Verify", service.CredentialCandidate, session.ID, csrfToken).Return(err)
					}
				}
			}
			if tt.call {
				switch tt.route {
				case "start":
					start.On("Execute", mock.Anything, candidateinterviewusecase.StartInterviewInput{ConsentID: consentID, IdempotencyKey: "start-1", ControllerID: controller}).Return(&candidateinterviewusecase.StartInterviewResult{Execution: facts, Created: tt.created}, tt.opErr)
				case "poll":
					poll.On("Execute", mock.Anything, candidateinterviewquery.GetExecutionInput{ExecutionID: executionID, ControllerID: controller}).Return(&facts, tt.opErr)
				case "takeover":
					takeover.On("Execute", mock.Anything, candidateinterviewusecase.TakeoverInput{ExecutionID: executionID, ControllerID: controller, ExpectedVersion: 1}).Return(&dto.TakeoverView{IsCurrentController: true, ControllerVersion: 2}, tt.opErr)
				case "cancel":
					facts.State = "CANCELLED"
					cancel.On("Execute", mock.Anything, candidateinterviewusecase.CancelInterviewInput{ExecutionID: executionID, ControllerID: controller, ExpectedVersion: 1}).Return(&facts, tt.opErr)
				}
			}
			path := "/candidate/executions"
			method := http.MethodPost
			if tt.route != "start" {
				path += "/" + executionID
				if tt.route == "poll" {
					method = http.MethodGet
				} else {
					path += "/" + tt.route
				}
			}
			if tt.method != "" {
				method = tt.method
			}
			rec, view := do(t, h, method, path, tt.body, func(r *http.Request) {
				executionHeaders(r)
				if tt.mutate != nil {
					tt.mutate(r)
				}
			})
			if tt.code != "" {
				require.Equal(t, tt.code, errorCode(t, rec, view, tt.status))
				require.NotContains(t, rec.Body.String(), "hunter2")
				return
			}
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			require.Equal(t, "success", view.Status)
			if tt.route == "takeover" {
				require.Equal(t, map[string]any{"isCurrentController": true, "controllerVersion": float64(2)}, view.Data)
			} else {
				require.Equal(t, map[string]any{"id": executionID, "state": facts.State, "version": float64(1), "hardDeadline": nil, "isCurrentController": true, "controllerVersion": float64(1)}, view.Data)
			}
		})
	}
}
