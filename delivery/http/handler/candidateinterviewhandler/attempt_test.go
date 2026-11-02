package candidateinterviewhandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// attemptCase describes one session-attempt or readiness transport scenario.
type attemptCase struct {
	name     string
	ready    bool
	method   string
	body     string
	mutate   func(*http.Request)
	session  string
	call     bool
	replayed bool
	opErr    error
	status   int
	code     string
}

// attemptRequest is a valid session-attempt body.
const attemptRequest = `{"sdp":"v=0","intent":"activate","expectedVersion":4,"controllerId":"tab-1","idempotencyKey":"attempt-key-1"}`

// readyRequest is a valid readiness body.
const readyRequest = `{"expectedVersion":5,"controllerId":"tab-1"}`

// TestSessionAttemptTransport covers the session-attempt and ready routes: validation order, CSRF, replay status and safe errors.
func TestSessionAttemptTransport(t *testing.T) {
	tests := []attemptCase{
		{name: "attempt created", body: attemptRequest, session: "ok", call: true, status: 201},
		{name: "attempt replayed", body: attemptRequest, session: "ok", call: true, replayed: true, status: 200},
		{name: "ready", ready: true, body: readyRequest, session: "ok", call: true, status: 200},
		{name: "attempt conflict", body: attemptRequest, session: "ok", call: true, opErr: interviewusecase.ErrAttemptConflict, status: 409, code: errs.IRTConflict},
		{name: "ready conflict", ready: true, body: readyRequest, session: "ok", call: true, opErr: interviewusecase.ErrAttemptConflict, status: 409, code: errs.IRTConflict},
		{name: "attempt not ready", body: attemptRequest, session: "ok", call: true, opErr: interviewusecase.ErrAttemptNotReady, status: 409, code: errs.IRTNotReady},
		{name: "ready not current", ready: true, body: readyRequest, session: "ok", call: true, opErr: interviewusecase.ErrAttemptNotReady, status: 409, code: errs.IRTNotReady},
		{name: "foreign execution attempt", body: attemptRequest, session: "ok", call: true, opErr: service.ErrInterviewNotFound, status: 404, code: errs.NotFound},
		{name: "foreign execution ready", ready: true, body: readyRequest, session: "ok", call: true, opErr: service.ErrInterviewNotFound, status: 404, code: errs.NotFound},
		{name: "provider unavailable", body: attemptRequest, session: "ok", call: true, opErr: interviewusecase.ErrProviderUnavailable, status: 503, code: errs.SessionUnavailable},
		{name: "operation rejects input", body: attemptRequest, session: "ok", call: true, opErr: interviewusecase.ErrAttemptInvalid, status: 400, code: errs.InvalidInput},
		{name: "unbound session", body: attemptRequest, session: "ok", call: true, opErr: service.ErrCandidateSessionUnbound, status: 401, code: errs.IAUnauthenticated},
		{name: "technical safe", body: attemptRequest, session: "ok", call: true, opErr: errors.New("password=hunter2"), status: 500, code: errs.InternalError},
		{name: "attempt wrong method", method: "GET", status: 405, code: errs.MethodNotAllowed},
		{name: "ready wrong method", ready: true, method: "GET", status: 405, code: errs.MethodNotAllowed},
		{name: "blank offer", body: `{"sdp":"  ","intent":"activate","expectedVersion":4,"controllerId":"tab-1","idempotencyKey":"attempt-key-1"}`, status: 400, code: errs.InvalidInput},
		{name: "unknown intent", body: `{"sdp":"v=0","intent":"restart","expectedVersion":4,"controllerId":"tab-1","idempotencyKey":"attempt-key-1"}`, status: 400, code: errs.InvalidInput},
		{name: "missing key", body: `{"sdp":"v=0","intent":"activate","expectedVersion":4,"controllerId":"tab-1"}`, status: 400, code: errs.InvalidInput},
		{name: "bad key", body: `{"sdp":"v=0","intent":"activate","expectedVersion":4,"controllerId":"tab-1","idempotencyKey":" bad"}`, status: 400, code: errs.InvalidInput},
		{name: "missing controller", body: `{"sdp":"v=0","intent":"activate","expectedVersion":4,"idempotencyKey":"attempt-key-1"}`, status: 400, code: errs.InvalidInput},
		{name: "zero version", body: `{"sdp":"v=0","intent":"activate","expectedVersion":0,"controllerId":"tab-1","idempotencyKey":"attempt-key-1"}`, status: 400, code: errs.InvalidInput},
		{name: "unknown field", body: `{"sdp":"v=0","intent":"activate","expectedVersion":4,"controllerId":"tab-1","idempotencyKey":"attempt-key-1","stage":"closing"}`, status: 400, code: errs.InvalidInput},
		{name: "duplicate field", ready: true, body: `{"expectedVersion":5,"expectedVersion":6,"controllerId":"tab-1"}`, status: 400, code: errs.InvalidInput},
		{name: "ready missing controller", ready: true, body: `{"expectedVersion":5}`, status: 400, code: errs.InvalidInput},
		{name: "ready zero version", ready: true, body: `{"expectedVersion":0,"controllerId":"tab-1"}`, status: 400, code: errs.InvalidInput},
		{name: "content type before session", body: attemptRequest, mutate: func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, status: 400, code: errs.InvalidInput},
		{name: "origin before session", body: attemptRequest, mutate: func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, status: 400, code: errs.InvalidInput},
		{name: "missing cookie", body: attemptRequest, mutate: func(r *http.Request) { r.Header.Del("Cookie") }, status: 401, code: errs.IAUnauthenticated},
		{name: "missing csrf", body: attemptRequest, session: "no-csrf", mutate: func(r *http.Request) { r.Header.Del("X-CSRF-Token") }, status: 403, code: errs.CINForbidden},
		{name: "bad csrf ready", ready: true, body: readyRequest, session: "bad-csrf", status: 403, code: errs.CINForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, g := newGraph(t)
			attempt, ready := mocks.NewAttemptOperation(t), mocks.NewReadyOperation(t)
			h.Attempt, h.Ready = attempt, ready
			if tt.session != "" {
				session := testdata.CandidateSessionRecord(t)
				g.current.On("Resolve", mock.Anything).Return(session, nil)
				if tt.session != "no-csrf" {
					var err error
					if tt.session == "bad-csrf" {
						err = service.ErrCSRFMismatch
					}
					g.csrf.On("Verify", service.CredentialCandidate, session.ID, csrfToken).Return(err)
				}
			}
			if tt.call && tt.ready {
				ready.On("Execute", mock.Anything, interviewusecase.ReadyInput{ExecutionID: executionID, AttemptID: "attempt-1", ControllerID: controller, ExpectedVersion: 5}).
					Return(&interviewusecase.ReadyResult{AttemptID: "attempt-1", SessionID: "session-1", ExecutionVersion: 5, SessionCondition: "ready"}, tt.opErr)
			}
			if tt.call && !tt.ready {
				attempt.On("Execute", mock.Anything, interviewusecase.AttemptInput{ExecutionID: executionID, ControllerID: controller, IdempotencyKey: "attempt-key-1", Intent: "activate", SDP: "v=0", ExpectedVersion: 4}).
					Return(&interviewusecase.AttemptResult{AttemptID: "attempt-1", SessionID: "session-1", SDP: "answer", ExecutionVersion: 5, SessionCondition: "pending", Replayed: tt.replayed}, tt.opErr)
			}
			path := "/candidate/executions/" + executionID + "/session-attempts"
			if tt.ready {
				path += "/attempt-1/ready"
			}
			method := http.MethodPost
			if tt.method != "" {
				method = tt.method
			}
			rec, view := do(t, h, method, path, tt.body, func(r *http.Request) {
				withCSRF(r)
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
			if tt.ready {
				require.Equal(t, map[string]any{"attemptId": "attempt-1", "sessionId": "session-1", "executionVersion": float64(5), "sessionCondition": "ready"}, view.Data)
				return
			}
			require.Equal(t, map[string]any{"attemptId": "attempt-1", "sessionId": "session-1", "sdp": "answer", "executionVersion": float64(5), "sessionCondition": "pending"}, view.Data)
		})
	}
}
