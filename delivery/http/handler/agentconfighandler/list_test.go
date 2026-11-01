package agentconfighandler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// listHTTPCase names one instance-collection HTTP outcome.
type listHTTPCase struct {
	name          string
	method        string
	authenticated bool
	cause         error
	wantStatus    int
	wantCode      string
	wantEmpty     bool
	wantFiltered  bool
	wantCreate    bool
}

// TestListAgentInstancesHTTP covers HR listing, denial, create, and unsupported methods.
func TestListAgentInstancesHTTP(t *testing.T) {
	tests := []listHTTPCase{
		{name: "hr empty list", method: http.MethodGet, authenticated: true, wantStatus: http.StatusOK, wantEmpty: true},
		{name: "hr filtered instance", method: http.MethodGet, authenticated: true, wantStatus: http.StatusOK, wantFiltered: true},
		{name: "admin forbidden", method: http.MethodGet, authenticated: true, cause: service.ErrForbidden, wantStatus: http.StatusForbidden, wantCode: errs.ACPForbidden},
		{name: "unauthenticated", method: http.MethodGet, wantStatus: http.StatusUnauthorized, wantCode: errs.ACPUnauthenticated},
		{name: "post create unchanged", method: http.MethodPost, authenticated: true, wantStatus: http.StatusCreated, wantCreate: true},
		{name: "put method not allowed", method: http.MethodPut, authenticated: true, wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, create, _, _, _, _, _, list, csrf, sessions := newHandler(t, httpOrigin, false)
			if tt.wantCreate {
				authorizeMutation(t, sessions, csrf)
				create.On("Execute", mock.Anything, mock.MatchedBy(func(in agentconfigusecase.CreateInput) bool {
					return in.TemplateID == catalog.TemplateHRInterviewer
				})).Return(sampleHRInstance(), nil).Once()
			}
			if tt.method == http.MethodGet && tt.authenticated {
				switch {
				case tt.cause != nil:
					list.On("Execute", mock.Anything).Return(nil, tt.cause).Once()
				case tt.wantFiltered:
					list.On("Execute", mock.Anything).Return([]dto.AgentInstance{*sampleHRInstance()}, nil).Once()
				default:
					list.On("Execute", mock.Anything).Return([]dto.AgentInstance{}, nil).Once()
				}
			}
			var cookie *http.Cookie
			if tt.authenticated {
				cookie = adminCookie(false, sessionTok)
			}
			body := ""
			csrfHeader := ""
			if tt.method == http.MethodPost {
				body = createHRBody
				csrfHeader = csrfTok
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request(tt.method, "/admin/agent-instances", httpOrigin, body, csrfHeader, cookie, nil))
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			view := decodeEnvelope(t, rec)
			if tt.wantCode != "" {
				require.Equal(t, "error", view.Status)
				require.Equal(t, tt.wantCode, view.Error.Code)
				return
			}
			require.Equal(t, "success", view.Status)
			if tt.wantEmpty {
				require.Equal(t, "[]", strings.TrimSpace(string(view.Data)))
				require.NotEqual(t, "null", strings.TrimSpace(string(view.Data)))
			}
			if tt.wantFiltered {
				require.Contains(t, string(view.Data), `"salaryCeilingConfigured":true`)
				require.NotContains(t, rec.Body.String(), salaryRefID)
			}
			if tt.wantCreate {
				require.Contains(t, string(view.Data), `"id":"`+instanceID+`"`)
			}
			if tt.wantEmpty {
				var items []json.RawMessage
				require.NoError(t, json.Unmarshal(view.Data, &items))
				require.NotNil(t, items)
				require.Empty(t, items)
			}
		})
	}
}
