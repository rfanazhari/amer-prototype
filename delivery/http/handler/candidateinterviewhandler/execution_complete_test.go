package candidateinterviewhandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCompleteTransport proves the finish route maps the operation outcome and validates input before it runs.
func TestCompleteTransport(t *testing.T) {
	body := `{"expectedVersion":1}`
	// completeCase describes one finish request, whether the operation runs, and the expected response.
	type completeCase struct {
		name    string
		body    string
		method  string
		mutate  func(*http.Request)
		session string
		call    bool
		opErr   error
		status  int
		code    string
	}
	tests := []completeCase{
		{name: "completes", body: body, session: "ok", call: true, status: 200},
		{name: "not the controller", body: body, session: "ok", call: true, opErr: candidateinterviewusecase.ErrNotController, status: 403, code: errs.CINNotController},
		{name: "terminal", body: body, session: "ok", call: true, opErr: candidateinterview.ErrExecutionTerminal, status: 409, code: errs.CINTerminal},
		{name: "not live", body: body, session: "ok", call: true, opErr: service.ErrInterviewNotLive, status: 409, code: errs.CINNotLive},
		{name: "stale version", body: body, session: "ok", call: true, opErr: service.ErrInterviewConflict, status: 409, code: errs.CINControllerConflict},
		{name: "foreign execution", body: body, session: "ok", call: true, opErr: service.ErrInterviewNotFound, status: 404, code: errs.NotFound},
		{name: "unbound lifecycle", body: body, session: "ok", call: true, opErr: service.ErrInterviewDependencyUnavailable, status: 503, code: errs.CINDependencyUnavailable},
		{name: "technical failure stays safe", body: body, session: "ok", call: true, opErr: errors.New("password=hunter2"), status: 500, code: errs.InternalError},
		{name: "wrong method", body: body, method: http.MethodGet, status: 405, code: errs.MethodNotAllowed},
		{name: "missing version", body: `{}`, status: 400, code: errs.InvalidInput},
		{name: "unknown field", body: `{"expectedVersion":1,"id":"x"}`, status: 400, code: errs.InvalidInput},
		{name: "invalid tab", body: body, mutate: func(r *http.Request) { r.Header.Set("X-Interview-Controller", " bad") }, status: 400, code: errs.InvalidInput},
		{name: "missing session", body: body, mutate: func(r *http.Request) { r.Header.Del("Cookie") }, status: 401, code: errs.IAUnauthenticated},
		{name: "missing csrf", body: body, session: "no-csrf", mutate: func(r *http.Request) { r.Header.Del("X-CSRF-Token") }, status: 403, code: errs.CINForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, g := newGraph(t)
			complete := mocks.NewCompleteInterviewOperation(t)
			h.Complete = complete
			facts := dto.InterviewExecutionView{ID: executionID, State: "COMPLETED", Version: 4, IsCurrentController: true, ControllerVersion: 2}
			if tt.session != "" {
				session := testdata.CandidateSessionRecord(t)
				g.current.On("Resolve", mock.Anything).Return(session, nil)
				if tt.session != "no-csrf" {
					g.csrf.On("Verify", service.CredentialCandidate, session.ID, csrfToken).Return(nil)
				}
			}
			if tt.call {
				complete.On("Execute", mock.Anything, candidateinterviewusecase.CompleteInterviewInput{ExecutionID: executionID, ControllerID: controller, ExpectedVersion: 1}).Return(&facts, tt.opErr)
			}
			method := http.MethodPost
			if tt.method != "" {
				method = tt.method
			}

			rec, view := do(t, h, method, "/candidate/executions/"+executionID+"/complete", tt.body, func(r *http.Request) {
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
			require.Equal(t, map[string]any{"id": executionID, "state": "COMPLETED", "version": float64(4), "hardDeadline": nil, "isCurrentController": true, "controllerVersion": float64(2)}, view.Data)
		})
	}
}
