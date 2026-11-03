package runtime

import (
	"context"
	"net/http"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/candidateinterview"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/candidateinterviewhandler"
	domain "github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/candidateinterviewrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/executionrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/agentreference"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/candidatelifecycle"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/interviewlifecycle"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/interviewvoice"
	"gorm.io/gorm"
)

// NewCandidateInterview composes the candidate interview JSON handler and page shell over the shared pool.
// Execution Lifecycle and owned runtime progress join the same execution-locked transaction context.
func NewCandidateInterview(db *gorm.DB, cfg Config, logs *logmanager.Application) (http.Handler, http.Handler) {
	timer := clock.System{}
	current := accessauthority.CandidateSessions{
		Store:  candidateinterviewrepository.Sessions{DB: db},
		Tokens: accesscredentials.Tokens{},
		Clock:  timer,
	}
	consents := candidateinterviewrepository.ConsentStore{DB: db, Clock: timer}
	controllers := candidateinterviewrepository.ControllerStore{DB: db}
	executions := executionrepository.Store{DB: db}
	lifecycle := candidatelifecycle.Lifecycle{Assignments: executions, Transactions: executions, Executions: executions, Commands: executions, Clock: timer}
	progress := interviewrepository.Store{DB: db, Pins: interviewrepository.Pins{DB: db}, Clock: timer}
	commands := interviewrepository.Commands{DB: db, Clock: timer}
	authority := interviewlifecycle.Lifecycle{Transactions: executions, Executions: executions, Commands: executions, Attempts: executionrepository.Ledger{DB: db}, Progress: progress, Clock: timer}
	driver := &interviewvoice.Driver{APIKey: cfg.OpenAIAPIKey, Model: cfg.OpenAILiveModel, Voice: cfg.OpenAILiveVoice, SessionsURL: cfg.OpenAILiveURL, AttachURL: cfg.OpenAILiveAttachURL}
	workflow := interviewusecase.Workflow{Transactions: executions, Commands: commands, Lifecycle: authority, Store: progress, Driver: driver, Clock: timer, References: agentreference.Local{}, ProviderReady: func() bool { return cfg.OpenAIAPIKey != "" }}
	catalog := approvedNoticeCatalog{notice: CurrentCandidateNotice()}
	return candidateRuntime{driver: driver, Handler: candidateinterviewhandler.Handler{
		Attempt: interviewusecase.Attempts{Current: current, Executions: lifecycle, Controllers: controllers, Commands: commands, Revisions: interviewrepository.Revisions{DB: db}, Progress: progress, Sessions: workflow, Clock: timer},
		Ready:   interviewusecase.Ready{Current: current, Executions: lifecycle, Controllers: controllers, Lifecycle: authority, Progress: progress, Clock: timer},
		Get: candidateinterviewquery.GetInterview{
			Current: current, Catalog: catalog, Consents: consents,
			Controllers: controllers, Lifecycle: lifecycle, Clock: timer,
		},
		GetExecution: candidateinterviewquery.GetExecution{Current: current, Controllers: controllers, Lifecycle: lifecycle},
		Start:        candidateinterviewusecase.StartInterview{Current: current, Catalog: catalog, Consents: consents, Controllers: controllers, Lifecycle: lifecycle, Clock: timer},
		Takeover:     candidateinterviewusecase.Takeover{Current: current, Controllers: controllers, Lifecycle: lifecycle, Clock: timer},
		Cancel:       candidateinterviewusecase.CancelInterview{Current: current, Controllers: controllers, Lifecycle: lifecycle},
		Complete:     candidateinterviewusecase.CompleteInterview{Current: current, Controllers: controllers, Lifecycle: lifecycle, Completer: workflow},
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
	}}, candidateinterview.Handler{}
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

// candidateRuntime owns provider sideband sockets for the composed interview routes.
type candidateRuntime struct {
	http.Handler
	driver *interviewvoice.Driver
}

// Close releases all sidebands after HTTP draining, without treating close as SQL rollback.
func (h candidateRuntime) Close() error { return h.driver.Shutdown() }
