package runtime

import (
	"context"
	"net/http"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/candidateinterview"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/candidateinterviewhandler"
	domain "github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/candidateinterviewrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"gorm.io/gorm"
)

// NewCandidateInterview composes the candidate interview JSON handler and page shell over the shared pool.
// Execution Lifecycle and the controller store have no adapter yet, so they are bound to fail-closed stand-ins.
func NewCandidateInterview(db *gorm.DB, cfg Config, logs *logmanager.Application) (http.Handler, http.Handler) {
	store := accessrepository.Store{DB: db}
	timer := clock.System{}
	current := accessauthority.CandidateSessions{
		Store:  accessrepository.CandidateSessionSecrets{Store: store},
		Tokens: accesscredentials.Tokens{},
		Clock:  timer,
	}
	consents := candidateinterviewrepository.ConsentStore{DB: db, Clock: timer}
	catalog := approvedNoticeCatalog{notice: CurrentCandidateNotice()}
	return candidateinterviewhandler.Handler{
		Get: candidateinterviewquery.GetInterview{
			Current: current, Catalog: catalog, Consents: consents,
			Controllers: unboundControllers{}, Lifecycle: unboundLifecycle{}, Clock: timer,
		},
		Accept:       candidateinterviewusecase.AcceptNotice{Current: current, Catalog: catalog, Tx: consents, Consents: consents, Clock: timer},
		Current:      current,
		CSRF:         accesscredentials.CSRF{Secret: []byte(cfg.AccessCSRFSecret)},
		Origin:       cfg.AccessHTTPOrigin,
		SecureCookie: len(cfg.AccessHTTPOrigin) >= 8 && cfg.AccessHTTPOrigin[:8] == "https://",
		LogError: func(err error) {
			txn := logs.Start("", "request error", logmanager.TxnTypeOther)
			txn.NoticeError(err)
			txn.End()
		},
	}, candidateinterview.Handler{}
}

// approvedNoticeCatalog supplies the approved notice value; a value that fails the version rule counts as an empty catalog.
type approvedNoticeCatalog struct {
	notice CandidateNotice
}

var _ service.NoticeCatalog = approvedNoticeCatalog{}

// Current returns the approved notice, or false so acceptance fails closed.
func (c approvedNoticeCatalog) Current(context.Context) (service.Notice, bool) {
	version, err := domain.ParseNoticeVersion(c.notice.Version)
	if err != nil || c.notice.Statement == "" {
		return service.Notice{}, false
	}
	return service.Notice{Version: version, Statement: c.notice.Statement}, true
}

// unboundLifecycle stands in for Execution Lifecycle until that capability provides an adapter; it writes nothing.
type unboundLifecycle struct{}

var _ service.ExecutionLifecycle = unboundLifecycle{}

// CommitOrFind reports the dependency as unavailable.
func (unboundLifecycle) CommitOrFind(context.Context, service.CommitOrFindRequest) (service.CommitOrFindResult, error) {
	return service.CommitOrFindResult{}, service.ErrInterviewDependencyUnavailable
}

// Find reports the dependency as unavailable, which the interview read renders as no execution.
func (unboundLifecycle) Find(context.Context, string) (service.ExecutionFacts, error) {
	return service.ExecutionFacts{}, service.ErrInterviewDependencyUnavailable
}

// Cancel reports the dependency as unavailable.
func (unboundLifecycle) Cancel(context.Context, service.CancelRequest) (service.ExecutionFacts, error) {
	return service.ExecutionFacts{}, service.ErrInterviewDependencyUnavailable
}

// unboundControllers stands in for the controller store until takeover persistence exists; it writes nothing.
type unboundControllers struct{}

var _ candidateinterviewport.Controllers = unboundControllers{}

// Find reports that no controller is stored.
func (unboundControllers) Find(context.Context, string) (domain.ControllerSnapshot, error) {
	return domain.ControllerSnapshot{}, candidateinterviewport.ErrNotFound
}

// Claim reports the dependency as unavailable.
func (unboundControllers) Claim(context.Context, domain.ControllerSnapshot) error {
	return service.ErrInterviewDependencyUnavailable
}

// Update reports the dependency as unavailable.
func (unboundControllers) Update(context.Context, string, func(*domain.InterviewController) error) error {
	return service.ErrInterviewDependencyUnavailable
}
