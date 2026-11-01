package agentconfighandler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/agentconfighandler"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	httpOrigin   = "http://localhost:8080"
	httpsOrigin  = "https://app.example.test"
	sessionTok   = "plain-session-secret"
	csrfTok      = "signed-csrf-material"
	instanceID   = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8"
	revisionID   = "cccccccc-cccc-4ccc-8ccc-ccccccccccc8"
	validationID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb8"
	salaryRefID  = "22222222-2222-4222-8222-222222222222"
	publishKey   = "publish-invoice-007"
	createHRBody = `{"templateId":"hr-interviewer","templateVersion":"1","name":"Invoice reviewer","configuration":{}}`
)

// envelopeView decodes the public JSON envelope used by configuration HTTP assertions.
type envelopeView struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// newHandler wires generated operation mocks for one configuration HTTP scenario.
func newHandler(t *testing.T, origin string, secure bool) (agentconfighandler.Handler, *mocks.AgentConfigCatalogOperation, *mocks.AgentConfigCreateOperation, *mocks.AgentConfigInstanceOperation, *mocks.AgentConfigReplaceOperation, *mocks.AgentConfigValidateOperation, *mocks.AgentConfigPublishOperation, *mocks.AgentConfigRevisionOperation, *mocks.CSRFTokens, *mocks.CurrentAdministrativeSession) {
	t.Helper()
	catalogOp := mocks.NewAgentConfigCatalogOperation(t)
	create := mocks.NewAgentConfigCreateOperation(t)
	instance := mocks.NewAgentConfigInstanceOperation(t)
	replace := mocks.NewAgentConfigReplaceOperation(t)
	validate := mocks.NewAgentConfigValidateOperation(t)
	publish := mocks.NewAgentConfigPublishOperation(t)
	revision := mocks.NewAgentConfigRevisionOperation(t)
	csrf := mocks.NewCSRFTokens(t)
	sessions := mocks.NewCurrentAdministrativeSession(t)
	return agentconfighandler.Handler{
		Catalog: catalogOp, Create: create, Instance: instance, Replace: replace,
		Validate: validate, Publish: publish, Revision: revision, CSRF: csrf, Sessions: sessions,
		Origin: origin, SecureCookie: secure,
	}, catalogOp, create, instance, replace, validate, publish, revision, csrf, sessions
}

// request builds a configuration JSON request with origin, cookie, CSRF and optional extra headers.
func request(method, path, origin, body, csrf string, cookie *http.Cookie, extra http.Header) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
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
	for key, values := range extra {
		for _, value := range values {
			req.Header.Add(key, value)
		}
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

// authorizeMutation stubs a current administrative session and matching CSRF header.
func authorizeMutation(t *testing.T, sessions *mocks.CurrentAdministrativeSession, csrf *mocks.CSRFTokens) {
	t.Helper()
	session := testdata.AdministrativeSessionRecord(t)
	sessions.On("Resolve", mock.Anything).Return(session, testdata.AdministrativeAccountRecord(t), nil)
	csrf.On("Verify", service.CredentialAdministrative, session.ID, csrfTok).Return(nil)
}

// sampleCatalog returns both supported template identities without provider details.
func sampleCatalog() []dto.AgentTemplate {
	return []dto.AgentTemplate{
		{TemplateID: catalog.TemplateHRInterviewer, TemplateVersion: catalog.TemplateVersion1, BehaviorVersion: catalog.BehaviorVersion1, DisplayName: catalog.DisplayHRInterviewer, WarningPolicy: catalog.WarningPolicyInformational},
		{TemplateID: catalog.TemplateDocumentAnalyzer, TemplateVersion: catalog.TemplateVersion1, BehaviorVersion: catalog.BehaviorVersion1, DisplayName: catalog.DisplayDocumentAnalyzer, WarningPolicy: catalog.WarningPolicyInformational},
	}
}

// sampleHRInstance returns a confidentiality-filtered HR draft without an active revision.
func sampleHRInstance() *dto.AgentInstance {
	return &dto.AgentInstance{
		ID: instanceID, Name: "Invoice reviewer", TemplateID: catalog.TemplateHRInterviewer,
		TemplateVersion: catalog.TemplateVersion1, BehaviorVersion: catalog.BehaviorVersion1, DraftVersion: 1,
		Configuration: dto.AgentConfigurationView{HR: &dto.HRConfigurationView{
			MandatoryQuestions: []dto.MandatoryQuestionView{}, CompetencyTopics: []string{},
			CompensationPolicy: &dto.CompensationPolicyView{DiscussionMode: "expectations_only", SalaryCeilingConfigured: true},
		}},
		CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	}
}

// sampleDocumentInstance returns a Document Analyzer draft view.
func sampleDocumentInstance() *dto.AgentInstance {
	return &dto.AgentInstance{
		ID: instanceID, Name: "Invoice reviewer", TemplateID: catalog.TemplateDocumentAnalyzer,
		TemplateVersion: catalog.TemplateVersion1, BehaviorVersion: catalog.BehaviorVersion1, DraftVersion: 1,
		Configuration: dto.AgentConfigurationView{Document: &dto.DocumentConfigurationView{RequiredFields: []string{"invoice_number"}}},
		CreatedAt:     testdata.Now(), UpdatedAt: testdata.Now(),
	}
}

// sampleValidation returns a persisted assessment, including blocking findings when ineligible.
func sampleValidation(eligible bool) *dto.AgentValidation {
	findings := []dto.AgentFinding{}
	if !eligible {
		findings = []dto.AgentFinding{{Severity: "error", Code: catalog.CodeRequiredFieldsEmpty, Path: "/requiredFields", Message: "at least one required invoice field must be selected"}}
	}
	return &dto.AgentValidation{
		ValidationID: validationID, InstanceID: instanceID, TemplateID: catalog.TemplateDocumentAnalyzer,
		TemplateVersion: catalog.TemplateVersion1, BehaviorVersion: catalog.BehaviorVersion1, DraftVersion: 1,
		Eligible: eligible, Findings: findings, CreatedAt: testdata.Now(),
	}
}

// sampleRevision returns a confidentiality-filtered published revision.
func sampleRevision() *dto.AgentRevision {
	return &dto.AgentRevision{
		RevisionID: revisionID, InstanceID: instanceID, ValidationID: validationID,
		TemplateID: catalog.TemplateHRInterviewer, TemplateVersion: catalog.TemplateVersion1, BehaviorVersion: catalog.BehaviorVersion1,
		RevisionNumber: 1, SourceDraftVersion: 3,
		Configuration: dto.AgentConfigurationView{HR: &dto.HRConfigurationView{
			CompensationPolicy: &dto.CompensationPolicyView{DiscussionMode: "expectations_only", SalaryCeilingConfigured: true},
		}},
		PublishedAt: testdata.Now(),
	}
}
