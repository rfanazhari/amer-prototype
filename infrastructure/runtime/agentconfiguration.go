package runtime

import (
	"strings"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/query/agentconfigquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/agentconfighandler"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/agentreference"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"gorm.io/gorm"
)

// assembleAgentConfiguration wires secured configuration operations onto the existing administrative authority adapter.
func assembleAgentConfiguration(db *gorm.DB, cfg Config, logs *logmanager.Application) agentconfighandler.Handler {
	store := agentconfigrepository.Store{DB: db}
	accessStore := accessrepository.Store{DB: db}
	timer := clock.System{}
	tokens := accesscredentials.Tokens{}
	csrf := accesscredentials.CSRF{Secret: []byte(cfg.AccessCSRFSecret)}
	sessions := accessauthority.Sessions{Store: accessStore, Tokens: tokens, Clock: timer}
	authority := accessauthority.Authority{Sessions: sessions, Clock: timer}
	assessment := service.DraftAssessment{References: agentreference.Local{}}
	logErr := func(err error) {
		txn := logs.Start("", "request error", logmanager.TxnTypeOther)
		txn.NoticeError(err)
		txn.End()
	}
	return agentconfighandler.Handler{
		Catalog:       agentconfigquery.Catalog{Authority: authority},
		Create:        agentconfigusecase.Create{Authority: authority, Instances: store, Clock: timer},
		Instance:      agentconfigquery.Instance{Authority: authority, Reader: store},
		ListInstances: agentconfigquery.ListInstances{Authority: authority, Lister: store},
		Replace:       agentconfigusecase.Replace{Authority: authority, Instances: store, Clock: timer},
		Validate:      agentconfigusecase.Validate{Authority: authority, Reader: store, Instances: store, Assessment: assessment, Clock: timer},
		Publish:       agentconfigusecase.Publish{Authority: authority, Instances: store, Reader: store, Replay: store, Clock: timer},
		Revision:      agentconfigquery.Revision{Authority: authority, Reader: store},
		CSRF:          csrf,
		Sessions:      sessions,
		Origin:        cfg.AccessHTTPOrigin,
		SecureCookie:  strings.HasPrefix(cfg.AccessHTTPOrigin, "https://"),
		LogError:      logErr,
	}
}
