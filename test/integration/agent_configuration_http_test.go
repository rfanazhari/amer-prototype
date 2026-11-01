package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/query/agentconfigquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentbindingusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/agentconfighandler"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentbindingrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/agentreference"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	acpJobDescriptionID = "11111111-1111-4111-8111-111111111111"
	acpSalaryCeilingID  = "22222222-2222-4222-8222-222222222222"
	acpHRTopic          = "concurrency"
	acpHREditedTopic    = "distributed-systems"
)

// configHTTP is the real administrative and configuration HTTP graph on one PostgreSQL pool.
type configHTTP struct {
	db     *gorm.DB
	clock  *shiftClock
	router http.Handler
}

// configAcceptanceCase names one backend configuration acceptance path.
type configAcceptanceCase struct {
	name string
	run  func(t *testing.T, graph configHTTP)
}

// staffSession is an ordinary login produced through the access routes.
type staffSession struct {
	cookie    *http.Cookie
	csrf      string
	accountID string
	role      string
}

// acpCatalogView is the catalog success payload.
type acpCatalogView struct {
	Templates []acpTemplateView `json:"templates"`
}

// acpTemplateView is one catalog template identity.
type acpTemplateView struct {
	TemplateID      string `json:"templateId"`
	TemplateVersion string `json:"templateVersion"`
	BehaviorVersion string `json:"behaviorVersion"`
	WarningPolicy   string `json:"warningPolicy"`
}

// acpInstanceView is the detached instance payload.
type acpInstanceView struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	TemplateID       string          `json:"templateId"`
	TemplateVersion  string          `json:"templateVersion"`
	BehaviorVersion  string          `json:"behaviorVersion"`
	DraftVersion     uint64          `json:"draftVersion"`
	Configuration    json.RawMessage `json:"configuration"`
	ActiveRevisionID *string         `json:"activeRevisionId"`
}

// acpValidationView is the persisted assessment payload.
type acpValidationView struct {
	ValidationID string       `json:"validationId"`
	InstanceID   string       `json:"instanceId"`
	DraftVersion uint64       `json:"draftVersion"`
	Eligible     bool         `json:"eligible"`
	Findings     []acpFinding `json:"findings"`
}

// acpFinding is one safe assessment finding.
type acpFinding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path"`
}

// acpRevisionView is the immutable revision payload.
type acpRevisionView struct {
	RevisionID         string          `json:"revisionId"`
	InstanceID         string          `json:"instanceId"`
	RevisionNumber     uint64          `json:"revisionNumber"`
	SourceDraftVersion uint64          `json:"sourceDraftVersion"`
	ValidationID       string          `json:"validationId"`
	TemplateID         string          `json:"templateId"`
	TemplateVersion    string          `json:"templateVersion"`
	BehaviorVersion    string          `json:"behaviorVersion"`
	Configuration      json.RawMessage `json:"configuration"`
}

// acpDocumentView is the document configuration object.
type acpDocumentView struct {
	RequiredFields []string `json:"requiredFields"`
}

// TestAgentConfigurationHTTPAcceptance proves both templates and the secured configuration routes on PostgreSQL.
// Fixture rows are created through ordinary HTTP operations. UI and assignment/execution consumers stay out of scope.
func TestAgentConfigurationHTTPAcceptance(t *testing.T) {
	tests := []configAcceptanceCase{
		{name: "hr interviewer create replace validate publish read", run: proveHRRoundTrip},
		{name: "document analyzer create replace validate publish read", run: proveDocumentRoundTrip},
		{name: "stale and foreign publication are refused", run: proveStalePublication},
		{name: "replay after a newer publication keeps the original revision", run: proveReplayAfterNewerPublication},
		{name: "another hr reads the company configuration", run: proveCrossHRRead},
		{name: "admin denial writes nothing", run: proveAdminDenied},
		{name: "candidate cookie is unauthenticated", run: proveCandidateCookie},
		{name: "exact session expiry is unauthenticated", run: proveExpiredSession},
		{name: "revoked session is unauthenticated", run: proveRevokedSession},
		{name: "missing csrf writes nothing", run: proveMissingCSRF},
		{name: "unknown template version is rejected", run: proveUnknownTemplate},
		{name: "blocking findings refuse publication", run: proveBlockedPublication},
		{name: "currency warning remains eligible", run: proveCurrencyWarning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, newConfigHTTP(t))
		})
	}
}

// newConfigHTTP wires the production adapters and dispatches configuration ahead of access.
func newConfigHTTP(t *testing.T) configHTTP {
	t.Helper()
	db := accessDB(t)
	clock := &shiftClock{at: time.Now().UTC().Truncate(time.Millisecond)}
	accessStore := accessrepository.Store{DB: db}
	configStore := agentconfigrepository.Store{DB: db}
	hasher := accesscredentials.Passwords{}
	tokens := accesscredentials.Tokens{}
	csrf := accesscredentials.CSRF{Secret: []byte(accessSecret)}
	sessions := accessauthority.Sessions{Store: accessStore, Tokens: tokens, Clock: clock}
	authority := accessauthority.Authority{Sessions: sessions, Clock: clock}
	candidates := accessrepository.CandidateIdentities{Store: accessStore}
	invitations := accessrepository.InvitationSecrets{Store: accessStore}
	candidateSecrets := accessrepository.CandidateSessionSecrets{Store: accessStore}
	candidateSessions := accessauthority.CandidateSessions{Store: candidateSecrets, Tokens: tokens, Clock: clock}
	accessHandler := accesshandler.Handler{
		Login:            accessusecase.Login{Accounts: accessStore, Issuance: accessStore, Hasher: hasher, Tokens: tokens, Clock: clock},
		Current:          accessquery.CurrentSession{Current: sessions, CSRF: csrf, Clock: clock},
		Provision:        accessusecase.ProvisionHR{Accounts: accessStore, Current: sessions, Hasher: hasher, Clock: clock},
		Logout:           accessusecase.Logout{Current: sessions, Sessions: accessStore, Clock: clock},
		CreateCandidate:  accessusecase.CreateCandidate{Candidates: candidates, Current: sessions, Clock: clock},
		CreateAssignment: agentbindingusecase.CreateAssignment{Current: sessions, Clock: clock, Candidates: candidates, Instances: configStore, Assignments: agentbindingrepository.Assignments{DB: db}},
		Candidate:        accessquery.Candidate{Candidates: candidates, Current: sessions, Clock: clock},
		ListCandidates:   accessquery.ListCandidates{Candidates: candidates, Current: sessions, Clock: clock},
		IssueInvitation:  accessusecase.IssueInvitation{Assignments: accessStore, Current: sessions, Tokens: tokens, Clock: clock},
		Exchange:         accessusecase.ExchangeInvitation{Invitations: invitations, Candidates: candidates, Assignments: accessStore, Tokens: tokens, Clock: clock},
		CandidateSession: accessquery.CandidateSession{Current: candidateSessions, Candidates: candidates, Assignments: accessStore, CSRF: csrf, Clock: clock},
		CSRF:             csrf,
		Sessions:         sessions,
		Origin:           accessOrigin,
		Limiter:          &accesshandler.LoginLimiter{},
		ExchangeLimiter:  &accesshandler.LoginLimiter{},
	}
	configHandler := agentconfighandler.Handler{
		Catalog:       agentconfigquery.Catalog{Authority: authority},
		Create:        agentconfigusecase.Create{Authority: authority, Instances: configStore, Clock: clock},
		Instance:      agentconfigquery.Instance{Authority: authority, Reader: configStore},
		ListInstances: agentconfigquery.ListInstances{Authority: authority, Lister: configStore},
		Replace:       agentconfigusecase.Replace{Authority: authority, Instances: configStore, Clock: clock},
		Validate:      agentconfigusecase.Validate{Authority: authority, Reader: configStore, Instances: configStore, Assessment: service.DraftAssessment{References: agentreference.Local{}}, Clock: clock},
		Publish:       agentconfigusecase.Publish{Authority: authority, Instances: configStore, Reader: configStore, Replay: configStore, Clock: clock},
		Revision:      agentconfigquery.Revision{Authority: authority, Reader: configStore},
		CSRF:          csrf,
		Sessions:      sessions,
		Origin:        accessOrigin,
	}
	return configHTTP{
		db: db, clock: clock,
		router: appruntime.Router{
			Access: accessHandler, Configuration: configHandler,
			Task: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unexpected task route", http.StatusTeapot)
			}),
		},
	}
}

// proveHRRoundTrip publishes an HR draft and keeps that revision after a later edit.
func proveHRRoundTrip(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	catalog := acpData[acpCatalogView](t, doACP(t, graph.router, http.MethodGet, "/admin/agent-templates", "", "", "", hr.cookie), http.StatusOK)
	require.Equal(t, "informational", templateByID(t, catalog, "hr-interviewer").WarningPolicy)
	require.Equal(t, "1", templateByID(t, catalog, "hr-interviewer").TemplateVersion)
	require.Equal(t, "1", templateByID(t, catalog, "document-analyzer").BehaviorVersion)
	name := uniqueLabel("Backend ")
	created := createInstance(t, graph, hr, "hr-interviewer", name, `{}`)
	require.Equal(t, uint64(1), created.DraftVersion)
	require.Nil(t, created.ActiveRevisionID)
	require.NotContains(t, string(created.Configuration), acpSalaryCeilingID)
	replaced := replaceInstance(t, graph, hr, created.ID, 1, hrConfiguration(acpHRTopic))
	require.Equal(t, uint64(2), replaced.DraftVersion)
	require.Contains(t, string(replaced.Configuration), `"salaryCeilingConfigured":true`)
	require.NotContains(t, string(replaced.Configuration), "salaryCeilingRef")
	validated := validateInstance(t, graph, hr, created.ID, 2)
	require.True(t, validated.Eligible)
	require.Empty(t, validated.Findings)
	key := commandKey()
	published := publishInstance(t, graph, hr, created.ID, 2, validated.ValidationID, key, http.StatusCreated)
	require.Equal(t, uint64(1), published.RevisionNumber)
	require.Equal(t, "hr-interviewer", published.TemplateID)
	require.Contains(t, string(published.Configuration), acpHRTopic)
	requireCommandScope(t, graph.db, key, hr.accountID, published.RevisionID)
	loaded := readRevision(t, graph, hr, created.ID, published.RevisionID)
	require.Equal(t, published.RevisionID, loaded.RevisionID)
	require.Contains(t, string(loaded.Configuration), acpHRTopic)
	current := readInstance(t, graph, hr, created.ID)
	require.NotNil(t, current.ActiveRevisionID)
	require.Equal(t, published.RevisionID, *current.ActiveRevisionID)
	edited := replaceInstance(t, graph, hr, created.ID, 2, hrConfiguration(acpHREditedTopic))
	require.Equal(t, uint64(3), edited.DraftVersion)
	preserved := readRevision(t, graph, hr, created.ID, published.RevisionID)
	require.Contains(t, string(preserved.Configuration), acpHRTopic)
	require.NotContains(t, string(preserved.Configuration), acpHREditedTopic)
	require.Equal(t, published.RevisionID, *readInstance(t, graph, hr, created.ID).ActiveRevisionID)
}

// proveDocumentRoundTrip publishes a document draft and keeps the original required fields.
func proveDocumentRoundTrip(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	name := uniqueLabel("Invoice ")
	created := createInstance(t, graph, hr, "document-analyzer", name, `{"requiredFields":[]}`)
	require.Equal(t, "document-analyzer", created.TemplateID)
	require.Equal(t, "1", created.TemplateVersion)
	require.Nil(t, created.ActiveRevisionID)
	replaced := replaceInstance(t, graph, hr, created.ID, 1, documentConfiguration("invoice_number", "total", "currency"))
	require.Equal(t, []string{"invoice_number", "total", "currency"}, documentFields(t, replaced.Configuration))
	validated := validateInstance(t, graph, hr, created.ID, 2)
	require.True(t, validated.Eligible)
	require.Empty(t, validated.Findings)
	key := commandKey()
	published := publishInstance(t, graph, hr, created.ID, 2, validated.ValidationID, key, http.StatusCreated)
	require.Equal(t, uint64(1), published.RevisionNumber)
	require.Equal(t, "1", published.BehaviorVersion)
	require.Equal(t, []string{"invoice_number", "total", "currency"}, documentFields(t, published.Configuration))
	requireCommandScope(t, graph.db, key, hr.accountID, published.RevisionID)
	require.Equal(t, published.RevisionID, readRevision(t, graph, hr, created.ID, published.RevisionID).RevisionID)
	edited := replaceInstance(t, graph, hr, created.ID, 2, documentConfiguration("currency"))
	require.Equal(t, uint64(3), edited.DraftVersion)
	require.Equal(t, []string{"invoice_number", "total", "currency"}, documentFields(t, readRevision(t, graph, hr, created.ID, published.RevisionID).Configuration))
	require.Equal(t, published.RevisionID, *readInstance(t, graph, hr, created.ID).ActiveRevisionID)
}

// proveStalePublication rejects an old draft version, a stale validation, and a foreign validation.
func proveStalePublication(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	created := createInstance(t, graph, hr, "hr-interviewer", uniqueLabel("Stale "), `{}`)
	replaceInstance(t, graph, hr, created.ID, 1, hrConfiguration(acpHRTopic))
	validated := validateInstance(t, graph, hr, created.ID, 2)
	replaceInstance(t, graph, hr, created.ID, 2, hrConfiguration(acpHREditedTopic))
	staleDraft := doACP(t, graph.router, http.MethodPost, publicationsPath(created.ID), publishBody(2, validated.ValidationID), hr.csrf, commandKey(), hr.cookie)
	requireCode(t, staleDraft, http.StatusConflict, errs.ACPDraftConflict)
	staleValidation := doACP(t, graph.router, http.MethodPost, publicationsPath(created.ID), publishBody(3, validated.ValidationID), hr.csrf, commandKey(), hr.cookie)
	requireCode(t, staleValidation, http.StatusConflict, errs.ACPValidationConflict)
	require.Zero(t, revisionCount(t, graph.db, created.ID))
	require.Nil(t, readInstance(t, graph, hr, created.ID).ActiveRevisionID)
	other := createInstance(t, graph, hr, "document-analyzer", uniqueLabel("Foreign "), `{"requiredFields":[]}`)
	foreign := doACP(t, graph.router, http.MethodPost, publicationsPath(other.ID), publishBody(1, validated.ValidationID), hr.csrf, commandKey(), hr.cookie)
	requireCode(t, foreign, http.StatusNotFound, errs.NotFound)
	require.Zero(t, revisionCount(t, graph.db, other.ID))
	require.Nil(t, readInstance(t, graph, hr, other.ID).ActiveRevisionID)
}

// proveReplayAfterNewerPublication returns the original revision and rejects a different intent for the same key.
func proveReplayAfterNewerPublication(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	created := createInstance(t, graph, hr, "hr-interviewer", uniqueLabel("Replay "), `{}`)
	replaceInstance(t, graph, hr, created.ID, 1, hrConfiguration(acpHRTopic))
	firstValidation := validateInstance(t, graph, hr, created.ID, 2)
	firstKey := commandKey()
	first := publishInstance(t, graph, hr, created.ID, 2, firstValidation.ValidationID, firstKey, http.StatusCreated)
	replaceInstance(t, graph, hr, created.ID, 2, hrConfiguration(acpHREditedTopic))
	secondValidation := validateInstance(t, graph, hr, created.ID, 3)
	second := publishInstance(t, graph, hr, created.ID, 3, secondValidation.ValidationID, commandKey(), http.StatusCreated)
	require.Equal(t, uint64(2), second.RevisionNumber)
	replay := publishInstance(t, graph, hr, created.ID, 2, firstValidation.ValidationID, firstKey, http.StatusOK)
	require.Equal(t, first.RevisionID, replay.RevisionID)
	require.Contains(t, string(replay.Configuration), acpHRTopic)
	require.NotContains(t, string(replay.Configuration), acpHREditedTopic)
	require.Equal(t, second.RevisionID, *readInstance(t, graph, hr, created.ID).ActiveRevisionID)
	conflict := doACP(t, graph.router, http.MethodPost, publicationsPath(created.ID), publishBody(3, secondValidation.ValidationID), hr.csrf, firstKey, hr.cookie)
	requireCode(t, conflict, http.StatusConflict, errs.ACPIdempotencyConflict)
	require.Equal(t, int64(2), revisionCount(t, graph.db, created.ID))
	require.Equal(t, second.RevisionID, *readInstance(t, graph, hr, created.ID).ActiveRevisionID)
	require.Contains(t, string(readRevision(t, graph, hr, created.ID, first.RevisionID).Configuration), acpHRTopic)
}

// proveCrossHRRead lets a second HR read another HR's company configuration and hides it from ADMIN and candidate cookies.
func proveCrossHRRead(t *testing.T, graph configHTTP) {
	t.Helper()
	owner := provisionHR(t, graph)
	other := provisionHR(t, graph)
	created := createInstance(t, graph, owner, "hr-interviewer", uniqueLabel("Shared "), `{}`)
	replaceInstance(t, graph, owner, created.ID, 1, hrConfiguration(acpHRTopic))
	validated := validateInstance(t, graph, owner, created.ID, 2)
	published := publishInstance(t, graph, owner, created.ID, 2, validated.ValidationID, commandKey(), http.StatusCreated)
	shared := readInstance(t, graph, other, created.ID)
	require.Equal(t, created.ID, shared.ID)
	require.Contains(t, string(shared.Configuration), `"salaryCeilingConfigured":true`)
	require.NotContains(t, string(shared.Configuration), acpSalaryCeilingID)
	listed := listedInstance(t, listInstances(t, graph, other), created.ID)
	require.Contains(t, string(listed.Configuration), `"salaryCeilingConfigured":true`)
	require.Equal(t, created.ID, listedInstance(t, listInstances(t, graph, owner), created.ID).ID)
	require.Equal(t, published.RevisionID, readRevision(t, graph, other, created.ID, published.RevisionID).RevisionID)
	admin := loginAdminSession(t, graph)
	hidden := doACP(t, graph.router, http.MethodGet, instancePath(created.ID), "", "", "", admin.cookie)
	requireCode(t, hidden, http.StatusForbidden, errs.ACPForbidden)
	require.NotContains(t, hidden.Body.String(), acpHRTopic)
	candidate := doACP(t, graph.router, http.MethodGet, revisionsPath(created.ID, published.RevisionID), "", "", "", &http.Cookie{Name: "ia-candidate", Value: "candidate-token"})
	requireCode(t, candidate, http.StatusUnauthorized, errs.ACPUnauthenticated)
	require.NotContains(t, candidate.Body.String(), acpHRTopic)
}

// proveAdminDenied rejects configuration reads and writes for an ADMIN session.
func proveAdminDenied(t *testing.T, graph configHTTP) {
	t.Helper()
	admin := loginAdminSession(t, graph)
	require.Equal(t, "ADMIN", admin.role)
	catalog := doACP(t, graph.router, http.MethodGet, "/admin/agent-templates", "", "", "", admin.cookie)
	requireCode(t, catalog, http.StatusForbidden, errs.ACPForbidden)
	listed := doACP(t, graph.router, http.MethodGet, "/admin/agent-instances", "", "", "", admin.cookie)
	requireCode(t, listed, http.StatusForbidden, errs.ACPForbidden)
	name := uniqueLabel("Denied ")
	created := doACP(t, graph.router, http.MethodPost, "/admin/agent-instances", createBody("hr-interviewer", name, `{}`), admin.csrf, "", admin.cookie)
	requireCode(t, created, http.StatusForbidden, errs.ACPForbidden)
	requireNoNamedInstance(t, graph.db, name)
}

// proveCandidateCookie rejects a candidate cookie before configuration disclosure or writes.
func proveCandidateCookie(t *testing.T, graph configHTTP) {
	t.Helper()
	cookie := &http.Cookie{Name: "ia-candidate", Value: "candidate-token"}
	catalog := doACP(t, graph.router, http.MethodGet, "/admin/agent-templates", "", "", "", cookie)
	requireCode(t, catalog, http.StatusUnauthorized, errs.ACPUnauthenticated)
	listed := doACP(t, graph.router, http.MethodGet, "/admin/agent-instances", "", "", "", cookie)
	requireCode(t, listed, http.StatusUnauthorized, errs.ACPUnauthenticated)
	name := uniqueLabel("Candidate ")
	created := doACP(t, graph.router, http.MethodPost, "/admin/agent-instances", createBody("document-analyzer", name, `{"requiredFields":[]}`), "not-a-csrf-token", "", cookie)
	requireCode(t, created, http.StatusUnauthorized, errs.ACPUnauthenticated)
	requireNoNamedInstance(t, graph.db, name)
}

// proveExpiredSession refuses configuration routes at the exact administrative expiry.
func proveExpiredSession(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	require.Equal(t, http.StatusOK, doACP(t, graph.router, http.MethodGet, "/admin/agent-templates", "", "", "", hr.cookie).Code)
	graph.clock.set(graph.clock.Now().Add(2 * time.Hour))
	expired := doACP(t, graph.router, http.MethodGet, "/admin/agent-templates", "", "", "", hr.cookie)
	requireCode(t, expired, http.StatusUnauthorized, errs.ACPUnauthenticated)
	name := uniqueLabel("Expired ")
	denied := doACP(t, graph.router, http.MethodPost, "/admin/agent-instances", createBody("hr-interviewer", name, `{}`), hr.csrf, "", hr.cookie)
	requireCode(t, denied, http.StatusUnauthorized, errs.ACPUnauthenticated)
	requireNoNamedInstance(t, graph.db, name)
}

// proveRevokedSession refuses configuration routes after ordinary logout.
func proveRevokedSession(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	require.Equal(t, http.StatusOK, doACP(t, graph.router, http.MethodGet, "/admin/agent-templates", "", "", "", hr.cookie).Code)
	logout := doACP(t, graph.router, http.MethodDelete, "/admin/sessions/current", "", hr.csrf, "", hr.cookie)
	require.Equal(t, http.StatusOK, logout.Code, logout.Body.String())
	revoked := doACP(t, graph.router, http.MethodGet, "/admin/agent-templates", "", "", "", hr.cookie)
	requireCode(t, revoked, http.StatusUnauthorized, errs.ACPUnauthenticated)
	name := uniqueLabel("Revoked ")
	denied := doACP(t, graph.router, http.MethodPost, "/admin/agent-instances", createBody("hr-interviewer", name, `{}`), hr.csrf, "", hr.cookie)
	requireCode(t, denied, http.StatusUnauthorized, errs.ACPUnauthenticated)
	requireNoNamedInstance(t, graph.db, name)
}

// proveMissingCSRF rejects an authenticated write that omits the CSRF header.
func proveMissingCSRF(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	name := uniqueLabel("NoCSRF ")
	denied := doACP(t, graph.router, http.MethodPost, "/admin/agent-instances", createBody("hr-interviewer", name, `{}`), "", "", hr.cookie)
	requireCode(t, denied, http.StatusForbidden, errs.ACPForbidden)
	requireNoNamedInstance(t, graph.db, name)
}

// proveUnknownTemplate rejects a template version outside the catalog before insert.
func proveUnknownTemplate(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	name := uniqueLabel("Unknown ")
	denied := doACP(t, graph.router, http.MethodPost, "/admin/agent-instances", createBodyVersion("hr-interviewer", "9", name, `{}`), hr.csrf, "", hr.cookie)
	requireCode(t, denied, http.StatusBadRequest, errs.InvalidInput)
	requireNoNamedInstance(t, graph.db, name)
}

// proveBlockedPublication records incomplete findings and publishes no revision.
func proveBlockedPublication(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	created := createInstance(t, graph, hr, "document-analyzer", uniqueLabel("Blocked "), `{"requiredFields":[]}`)
	validated := validateInstance(t, graph, hr, created.ID, 1)
	require.False(t, validated.Eligible)
	require.True(t, hasFinding(validated, "error", "REQUIRED_FIELDS_EMPTY", "/requiredFields"))
	denied := doACP(t, graph.router, http.MethodPost, publicationsPath(created.ID), publishBody(1, validated.ValidationID), hr.csrf, commandKey(), hr.cookie)
	requireCode(t, denied, http.StatusUnprocessableEntity, errs.ACPPublicationBlocked)
	require.Zero(t, revisionCount(t, graph.db, created.ID))
	require.Nil(t, readInstance(t, graph, hr, created.ID).ActiveRevisionID)
}

// proveCurrencyWarning publishes a document draft that warns and still has no errors.
func proveCurrencyWarning(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	created := createInstance(t, graph, hr, "document-analyzer", uniqueLabel("Warning "), `{"requiredFields":[]}`)
	replaceInstance(t, graph, hr, created.ID, 1, documentConfiguration("total"))
	validated := validateInstance(t, graph, hr, created.ID, 2)
	require.True(t, validated.Eligible)
	require.True(t, hasFinding(validated, "warning", "CURRENCY_NOT_REQUIRED", "/requiredFields"))
	published := publishInstance(t, graph, hr, created.ID, 2, validated.ValidationID, commandKey(), http.StatusCreated)
	require.Equal(t, []string{"total"}, documentFields(t, published.Configuration))
	require.Equal(t, published.RevisionID, *readInstance(t, graph, hr, created.ID).ActiveRevisionID)
}

// provisionHR creates an HR account through bootstrap, provision and login.
func provisionHR(t *testing.T, graph configHTTP) staffSession {
	t.Helper()
	admin := loginAdminSession(t, graph)
	email := "hr-" + uuid.NewString() + "@example.com"
	body := `{"name":"Sinta Wijaya","email":"` + email + `","password":"` + accessPass + `"}`
	created := doACP(t, graph.router, http.MethodPost, "/admin/hr-accounts", body, admin.csrf, "", admin.cookie)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	return loginStaff(t, graph, email, "HR")
}

// loginAdminSession bootstraps one ADMIN and returns that ordinary session.
func loginAdminSession(t *testing.T, graph configHTTP) staffSession {
	t.Helper()
	email := "admin-" + uuid.NewString() + "@example.com"
	bootstrapAdmin(t, graph.db, graph.clock, email)
	return loginStaff(t, graph, email, "ADMIN")
}

// loginStaff logs in through the access route and reads the CSRF token.
func loginStaff(t *testing.T, graph configHTTP, email, role string) staffSession {
	t.Helper()
	logged := doACP(t, graph.router, http.MethodPost, "/admin/sessions", `{"email":"`+email+`","password":"`+accessPass+`"}`, "", "", nil)
	require.Equal(t, http.StatusCreated, logged.Code, logged.Body.String())
	cookie := cookieMap(logged)["ia-admin"]
	require.NotNil(t, cookie)
	current := doACP(t, graph.router, http.MethodGet, "/admin/session", "", "", "", cookie)
	require.Equal(t, http.StatusOK, current.Code, current.Body.String())
	var data struct {
		CSRFToken string `json:"csrfToken"`
		Account   struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"account"`
	}
	view := decodeAccess(t, current)
	require.NoError(t, json.Unmarshal(view.Data, &data))
	require.NotEmpty(t, data.CSRFToken)
	require.Equal(t, role, data.Account.Role)
	require.NotEmpty(t, data.Account.ID)
	return staffSession{cookie: cookie, csrf: data.CSRFToken, accountID: data.Account.ID, role: data.Account.Role}
}

// createInstance inserts one editable draft through POST /admin/agent-instances.
func createInstance(t *testing.T, graph configHTTP, session staffSession, templateID, name, configuration string) acpInstanceView {
	t.Helper()
	rec := doACP(t, graph.router, http.MethodPost, "/admin/agent-instances", createBody(templateID, name, configuration), session.csrf, "", session.cookie)
	view := acpData[acpInstanceView](t, rec, http.StatusCreated)
	require.Equal(t, name, view.Name)
	require.Equal(t, templateID, view.TemplateID)
	require.NotContains(t, rec.Body.String(), acpSalaryCeilingID)
	return view
}

// replaceInstance replaces a draft through PUT /admin/agent-instances/{id}/draft.
func replaceInstance(t *testing.T, graph configHTTP, session staffSession, id string, version uint64, configuration string) acpInstanceView {
	t.Helper()
	body := `{"expectedDraftVersion":` + u64(version) + `,"configuration":` + configuration + `}`
	rec := doACP(t, graph.router, http.MethodPut, instancePath(id)+"/draft", body, session.csrf, "", session.cookie)
	view := acpData[acpInstanceView](t, rec, http.StatusOK)
	require.NotContains(t, rec.Body.String(), acpSalaryCeilingID)
	return view
}

// validateInstance records an assessment through POST /admin/agent-instances/{id}/validations.
func validateInstance(t *testing.T, graph configHTTP, session staffSession, id string, version uint64) acpValidationView {
	t.Helper()
	body := `{"expectedDraftVersion":` + u64(version) + `}`
	rec := doACP(t, graph.router, http.MethodPost, instancePath(id)+"/validations", body, session.csrf, "", session.cookie)
	view := acpData[acpValidationView](t, rec, http.StatusCreated)
	require.Equal(t, id, view.InstanceID)
	require.Equal(t, version, view.DraftVersion)
	require.NotContains(t, rec.Body.String(), acpSalaryCeilingID)
	return view
}

// publishInstance publishes or replays through POST /admin/agent-instances/{id}/publications.
func publishInstance(t *testing.T, graph configHTTP, session staffSession, id string, version uint64, validationID, key string, status int) acpRevisionView {
	t.Helper()
	rec := doACP(t, graph.router, http.MethodPost, publicationsPath(id), publishBody(version, validationID), session.csrf, key, session.cookie)
	view := acpData[acpRevisionView](t, rec, status)
	require.Equal(t, id, view.InstanceID)
	require.Equal(t, validationID, view.ValidationID)
	require.NotContains(t, rec.Body.String(), acpSalaryCeilingID)
	require.NotContains(t, string(view.Configuration), "salaryCeilingRef")
	return view
}

// listInstances reads every instance through GET /admin/agent-instances.
func listInstances(t *testing.T, graph configHTTP, session staffSession) []acpInstanceView {
	t.Helper()
	rec := doACP(t, graph.router, http.MethodGet, "/admin/agent-instances", "", "", "", session.cookie)
	listed := acpData[[]acpInstanceView](t, rec, http.StatusOK)
	require.NotNil(t, listed)
	return listed
}

// listedInstance returns the collection item with id.
func listedInstance(t *testing.T, listed []acpInstanceView, id string) acpInstanceView {
	t.Helper()
	for _, item := range listed {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("instance %s missing from collection", id)
	return acpInstanceView{}
}

// readInstance reads one instance through GET /admin/agent-instances/{id}.
func readInstance(t *testing.T, graph configHTTP, session staffSession, id string) acpInstanceView {
	t.Helper()
	rec := doACP(t, graph.router, http.MethodGet, instancePath(id), "", "", "", session.cookie)
	view := acpData[acpInstanceView](t, rec, http.StatusOK)
	require.Equal(t, id, view.ID)
	require.NotContains(t, rec.Body.String(), acpSalaryCeilingID)
	return view
}

// readRevision reads one revision through GET /admin/agent-instances/{id}/revisions/{revision_id}.
func readRevision(t *testing.T, graph configHTTP, session staffSession, instanceID, revisionID string) acpRevisionView {
	t.Helper()
	rec := doACP(t, graph.router, http.MethodGet, revisionsPath(instanceID, revisionID), "", "", "", session.cookie)
	view := acpData[acpRevisionView](t, rec, http.StatusOK)
	require.Equal(t, revisionID, view.RevisionID)
	require.NotContains(t, string(view.Configuration), acpSalaryCeilingID)
	return view
}

// doACP performs one JSON request against the composed router.
func doACP(t *testing.T, handler http.Handler, method, path, body, csrf, idempotency string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.RemoteAddr = "192.0.2.80:9"
	req.Header.Set("Origin", accessOrigin)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if idempotency != "" {
		req.Header.Set("Idempotency-Key", idempotency)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// acpData decodes a success envelope into dest and rejects leaked ceiling identities.
func acpData[T any](t *testing.T, rec *httptest.ResponseRecorder, status int) T {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	view := decodeAccess(t, rec)
	require.Equal(t, "success", view.Status)
	require.Nil(t, view.Error)
	require.NotContains(t, rec.Body.String(), acpSalaryCeilingID)
	var dest T
	require.NoError(t, json.Unmarshal(view.Data, &dest))
	return dest
}

// requireCode asserts the public error status and code.
func requireCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	view := decodeAccess(t, rec)
	require.NotNil(t, view.Error)
	require.Equal(t, code, view.Error.Code)
	require.NotContains(t, rec.Body.String(), acpSalaryCeilingID)
}

// templateByID returns the catalog entry for templateID.
func templateByID(t *testing.T, catalog acpCatalogView, templateID string) acpTemplateView {
	t.Helper()
	for _, template := range catalog.Templates {
		if template.TemplateID == templateID {
			return template
		}
	}
	t.Fatalf("missing template %s", templateID)
	return acpTemplateView{}
}

// hasFinding reports whether findings contain the expected severity, code and path.
func hasFinding(view acpValidationView, severity, code, path string) bool {
	for _, finding := range view.Findings {
		if finding.Severity == severity && finding.Code == code && finding.Path == path {
			return true
		}
	}
	return false
}

// documentFields reads requiredFields from a document configuration object.
func documentFields(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var view acpDocumentView
	require.NoError(t, json.Unmarshal(raw, &view))
	return view.RequiredFields
}

// requireCommandScope checks the stored publication command uses the account UUID and original revision.
func requireCommandScope(t *testing.T, db *gorm.DB, key, accountID, revisionID string) {
	t.Helper()
	var scope, revision string
	err := db.WithContext(context.Background()).Raw(
		`SELECT scope::text, resulting_revision_id::text FROM agent_publication_commands WHERE request_key = ?`, key,
	).Row().Scan(&scope, &revision)
	require.NoError(t, err)
	require.Equal(t, accountID, scope)
	require.Equal(t, revisionID, revision)
}

// revisionCount returns persisted revisions for one instance.
func revisionCount(t *testing.T, db *gorm.DB, instanceID string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.WithContext(context.Background()).Raw(
		`SELECT COUNT(*) FROM agent_revisions WHERE instance_id = ?`, instanceID,
	).Scan(&count).Error)
	return count
}

// requireNoNamedInstance fails when a denied request inserted an instance with name.
func requireNoNamedInstance(t *testing.T, db *gorm.DB, name string) {
	t.Helper()
	var count int64
	require.NoError(t, db.WithContext(context.Background()).Raw(
		`SELECT COUNT(*) FROM agent_instances WHERE name = ?`, name,
	).Scan(&count).Error)
	require.Zero(t, count)
}

// hrConfiguration builds one publishable HR configuration for topic.
func hrConfiguration(topic string) string {
	return `{"jobDescriptionRef":{"id":"` + acpJobDescriptionID + `","version":"1"},"mandatoryQuestions":[{"id":"background","stage":"background","text":"Describe your backend experience."},{"id":"screening","stage":"screening","text":"How do you protect a concurrent update?"},{"id":"compensation","stage":"compensation","text":"What are your compensation expectations?"}],"competencyTopics":["` + topic + `"],"stageSeconds":{"background":60,"screening":240,"compensation":90,"closing":30},"compensationPolicy":{"discussionMode":"expectations_only","salaryCeilingRef":{"id":"` + acpSalaryCeilingID + `","version":"1"}}}`
}

// documentConfiguration builds a complete document configuration for fields.
func documentConfiguration(fields ...string) string {
	quoted := make([]string, len(fields))
	for i, field := range fields {
		quoted[i] = `"` + field + `"`
	}
	return `{"requiredFields":[` + strings.Join(quoted, ",") + `]}`
}

// createBody builds a version-1 create request.
func createBody(templateID, name, configuration string) string {
	return createBodyVersion(templateID, "1", name, configuration)
}

// createBodyVersion builds a create request for one template version.
func createBodyVersion(templateID, version, name, configuration string) string {
	return `{"templateId":"` + templateID + `","templateVersion":"` + version + `","name":"` + name + `","configuration":` + configuration + `}`
}

// publishBody builds a publication request.
func publishBody(version uint64, validationID string) string {
	return `{"expectedDraftVersion":` + u64(version) + `,"validationId":"` + validationID + `"}`
}

// instancePath returns the instance resource path.
func instancePath(id string) string {
	return "/admin/agent-instances/" + id
}

// publicationsPath returns the publication collection path.
func publicationsPath(id string) string {
	return instancePath(id) + "/publications"
}

// revisionsPath returns one revision resource path.
func revisionsPath(instanceID, revisionID string) string {
	return instancePath(instanceID) + "/revisions/" + revisionID
}

// commandKey returns one unused publication idempotency key.
func commandKey() string {
	return "k" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

// uniqueLabel returns a valid instance name that no other test inserts.
func uniqueLabel(prefix string) string {
	return prefix + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
}

// u64 renders a draft version as a JSON integer.
func u64(value uint64) string {
	return strconv.FormatUint(value, 10)
}
