package agentconfighandler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// operationHTTPCase names one successful configuration route against generated operation mocks.
type operationHTTPCase struct {
	name        string
	method      string
	path        string
	body        string
	idempotency string
	secure      bool
	origin      string
	wantStatus  int
	setup       func(*mocks.AgentConfigCatalogOperation, *mocks.AgentConfigCreateOperation, *mocks.AgentConfigInstanceOperation, *mocks.AgentConfigReplaceOperation, *mocks.AgentConfigValidateOperation, *mocks.AgentConfigPublishOperation, *mocks.AgentConfigRevisionOperation)
}

// TestSevenEndpointsReturnDetachedEnvelopes covers catalog, create, read, replace, validate, publish and revision.
func TestSevenEndpointsReturnDetachedEnvelopes(t *testing.T) {
	instancePath := "/admin/agent-instances/" + instanceID
	tests := []operationHTTPCase{
		{name: "catalog lists both templates", method: http.MethodGet, path: "/admin/agent-templates", origin: httpOrigin, wantStatus: http.StatusOK, setup: func(catalogOp *mocks.AgentConfigCatalogOperation, _ *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, _ *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			catalogOp.On("Execute", mock.Anything).Return(sampleCatalog(), nil).Once()
		}},
		{name: "create hr instance", method: http.MethodPost, path: "/admin/agent-instances", body: createHRBody, origin: httpOrigin, wantStatus: http.StatusCreated, setup: func(_ *mocks.AgentConfigCatalogOperation, create *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, _ *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			create.On("Execute", mock.Anything, mock.MatchedBy(func(in agentconfigusecase.CreateInput) bool {
				return in.TemplateID == catalog.TemplateHRInterviewer && in.Name == "Invoice reviewer" && in.HR != nil
			})).Return(sampleHRInstance(), nil).Once()
		}},
		{name: "create document instance", method: http.MethodPost, path: "/admin/agent-instances", body: `{"templateId":"document-analyzer","templateVersion":"1","name":"Invoice reviewer","configuration":{"requiredFields":["invoice_number"]}}`, origin: httpOrigin, wantStatus: http.StatusCreated, setup: func(_ *mocks.AgentConfigCatalogOperation, create *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, _ *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			create.On("Execute", mock.Anything, mock.MatchedBy(func(in agentconfigusecase.CreateInput) bool {
				return in.TemplateID == catalog.TemplateDocumentAnalyzer && len(in.DocumentFields) == 1
			})).Return(sampleDocumentInstance(), nil).Once()
		}},
		{name: "read instance", method: http.MethodGet, path: instancePath, origin: httpOrigin, wantStatus: http.StatusOK, setup: func(_ *mocks.AgentConfigCatalogOperation, _ *mocks.AgentConfigCreateOperation, instance *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, _ *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			instance.On("Execute", mock.Anything, instanceID).Return(sampleHRInstance(), nil).Once()
		}},
		{name: "replace draft", method: http.MethodPut, path: instancePath + "/draft", body: `{"expectedDraftVersion":1,"configuration":{"requiredFields":["invoice_number","total"]}}`, origin: httpOrigin, wantStatus: http.StatusOK, setup: func(_ *mocks.AgentConfigCatalogOperation, _ *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, replace *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, _ *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			replaced := sampleDocumentInstance()
			replaced.DraftVersion = 2
			replace.On("Execute", mock.Anything, mock.MatchedBy(func(in agentconfigusecase.ReplaceInput) bool {
				return in.InstanceID == instanceID && in.ExpectedDraftVersion == 1
			})).Return(replaced, nil).Once()
		}},
		{name: "validate returns 201 with errors", method: http.MethodPost, path: instancePath + "/validations", body: `{"expectedDraftVersion":1}`, origin: httpOrigin, wantStatus: http.StatusCreated, setup: func(_ *mocks.AgentConfigCatalogOperation, _ *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, validate *mocks.AgentConfigValidateOperation, _ *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			validate.On("Execute", mock.Anything, agentconfigusecase.ValidateInput{InstanceID: instanceID, ExpectedDraftVersion: 1}).Return(sampleValidation(false), nil).Once()
		}},
		{name: "publish new revision", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`, origin: httpOrigin, idempotency: publishKey, wantStatus: http.StatusCreated, setup: func(_ *mocks.AgentConfigCatalogOperation, _ *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, publish *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			publish.On("Execute", mock.Anything, agentconfigusecase.PublishInput{InstanceID: instanceID, ValidationID: validationID, IdempotencyKey: publishKey, ExpectedDraftVersion: 3}).
				Return(&agentconfigusecase.PublishResult{Revision: sampleRevision()}, nil).Once()
		}},
		{name: "publish replay", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`, origin: httpOrigin, idempotency: publishKey, wantStatus: http.StatusOK, setup: func(_ *mocks.AgentConfigCatalogOperation, _ *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, publish *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			publish.On("Execute", mock.Anything, agentconfigusecase.PublishInput{InstanceID: instanceID, ValidationID: validationID, IdempotencyKey: publishKey, ExpectedDraftVersion: 3}).
				Return(&agentconfigusecase.PublishResult{Revision: sampleRevision(), Replayed: true}, nil).Once()
		}},
		{name: "read revision", method: http.MethodGet, path: instancePath + "/revisions/" + revisionID, origin: httpOrigin, wantStatus: http.StatusOK, setup: func(_ *mocks.AgentConfigCatalogOperation, _ *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, _ *mocks.AgentConfigPublishOperation, revision *mocks.AgentConfigRevisionOperation) {
			revision.On("Execute", mock.Anything, instanceID, revisionID).Return(sampleRevision(), nil).Once()
		}},
		{name: "https host cookie", method: http.MethodGet, path: "/admin/agent-templates", origin: httpsOrigin, secure: true, wantStatus: http.StatusOK, setup: func(catalogOp *mocks.AgentConfigCatalogOperation, _ *mocks.AgentConfigCreateOperation, _ *mocks.AgentConfigInstanceOperation, _ *mocks.AgentConfigReplaceOperation, _ *mocks.AgentConfigValidateOperation, _ *mocks.AgentConfigPublishOperation, _ *mocks.AgentConfigRevisionOperation) {
			catalogOp.On("Execute", mock.Anything).Return(sampleCatalog(), nil).Once()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, catalogOp, create, instance, replace, validate, publish, revision, csrf, sessions := newHandler(t, tt.origin, tt.secure)
			if tt.method != http.MethodGet {
				authorizeMutation(t, sessions, csrf)
			}
			tt.setup(catalogOp, create, instance, replace, validate, publish, revision)
			extra := http.Header{}
			if tt.idempotency != "" {
				extra.Set("Idempotency-Key", tt.idempotency)
			}
			csrfHeader := ""
			if tt.method != http.MethodGet {
				csrfHeader = csrfTok
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request(tt.method, tt.path, tt.origin, tt.body, csrfHeader, adminCookie(tt.secure, sessionTok), extra))
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			view := decodeEnvelope(t, rec)
			require.Equal(t, "success", view.Status)
			require.NotContains(t, rec.Body.String(), salaryRefID)
			switch tt.name {
			case "catalog lists both templates", "https host cookie":
				require.Contains(t, string(view.Data), `"templateId":"hr-interviewer"`)
				require.Contains(t, string(view.Data), `"templateId":"document-analyzer"`)
				require.Contains(t, string(view.Data), `"configurationSchema"`)
				require.Contains(t, string(view.Data), `"warningPolicy":"informational"`)
			case "create hr instance", "read instance":
				require.Contains(t, string(view.Data), `"activeRevisionId":null`)
				require.Contains(t, string(view.Data), `"salaryCeilingConfigured":true`)
			case "validate returns 201 with errors":
				require.Contains(t, string(view.Data), `"eligible":false`)
				require.Contains(t, string(view.Data), `"REQUIRED_FIELDS_EMPTY"`)
			case "publish new revision", "publish replay", "read revision":
				require.Contains(t, string(view.Data), `"revisionId":"`+revisionID+`"`)
				require.Contains(t, string(view.Data), `"salaryCeilingConfigured":true`)
			}
		})
	}
}
