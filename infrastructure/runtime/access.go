package runtime

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"gorm.io/gorm"
)

// ValidateAccessSettings checks bootstrap identity, origin and CSRF secret without logging secrets.
func ValidateAccessSettings(cfg Config) error {
	if _, err := service.CanonicalAdministrativeName(cfg.AccessBootstrapAdminName); err != nil {
		return errors.New("ACCESS_BOOTSTRAP_ADMIN_NAME is invalid")
	}
	if _, err := service.CanonicalAdministrativeEmail(cfg.AccessBootstrapAdminEmail); err != nil {
		return errors.New("ACCESS_BOOTSTRAP_ADMIN_EMAIL is invalid")
	}
	if err := service.AcceptProvisionPassword(cfg.AccessBootstrapAdminPassword); err != nil {
		return errors.New("ACCESS_BOOTSTRAP_ADMIN_PASSWORD is invalid")
	}
	if err := validateAccessOrigin(cfg.AccessHTTPOrigin); err != nil {
		return err
	}
	if len(cfg.AccessCSRFSecret) < 32 {
		return errors.New("ACCESS_CSRF_SECRET is invalid")
	}
	return nil
}

// validateAccessOrigin accepts an exact https origin or http://localhost[:port] without a path.
func validateAccessOrigin(origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.HasSuffix(origin, "/") {
		return errors.New("ACCESS_HTTP_ORIGIN is invalid")
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if parsed.Hostname() == "localhost" {
			return nil
		}
	}
	return errors.New("ACCESS_HTTP_ORIGIN is invalid")
}

// bootstrapAccess inserts the first ADMIN when the administrative store is empty.
func bootstrapAccess(ctx context.Context, db *gorm.DB, cfg Config) error {
	return (accessusecase.Bootstrap{
		Accounts: accessrepository.Store{DB: db},
		Hasher:   accesscredentials.Passwords{},
		Clock:    clock.System{},
		Name:     cfg.AccessBootstrapAdminName,
		Email:    cfg.AccessBootstrapAdminEmail,
		Password: cfg.AccessBootstrapAdminPassword,
	}).Execute(ctx)
}

// assembleAccess wires administrative operations, cookies, CSRF and login rate bounding.
func assembleAccess(db *gorm.DB, cfg Config, logs *logmanager.Application) accesshandler.Handler {
	store := accessrepository.Store{DB: db}
	timer := clock.System{}
	hasher := accesscredentials.Passwords{}
	tokens := accesscredentials.Tokens{}
	csrf := accesscredentials.CSRF{Secret: []byte(cfg.AccessCSRFSecret)}
	sessions := accessauthority.Sessions{Store: store, Tokens: tokens, Clock: timer}
	logErr := func(err error) {
		txn := logs.Start("", "request error", logmanager.TxnTypeOther)
		txn.NoticeError(err)
		txn.End()
	}
	candidates := accessrepository.CandidateIdentities{Store: store}
	invitations := accessrepository.InvitationSecrets{Store: store}
	candidateSecrets := accessrepository.CandidateSessionSecrets{Store: store}
	candidateSessions := accessauthority.CandidateSessions{Store: candidateSecrets, Tokens: tokens, Clock: timer}
	return accesshandler.Handler{
		Login:            accessusecase.Login{Accounts: store, Issuance: store, Hasher: hasher, Tokens: tokens, Clock: timer},
		Current:          accessquery.CurrentSession{Current: sessions, CSRF: csrf, Clock: timer},
		Provision:        accessusecase.ProvisionHR{Accounts: store, Current: sessions, Hasher: hasher, Clock: timer},
		Logout:           accessusecase.Logout{Current: sessions, Sessions: store, Clock: timer},
		CreateCandidate:  accessusecase.CreateCandidate{Candidates: candidates, Current: sessions, Clock: timer},
		Candidate:        accessquery.Candidate{Candidates: candidates, Current: sessions, Clock: timer},
		IssueInvitation:  accessusecase.IssueInvitation{Assignments: store, Current: sessions, Tokens: tokens, Clock: timer},
		Exchange:         accessusecase.ExchangeInvitation{Invitations: invitations, Candidates: candidates, Assignments: store, Tokens: tokens, Clock: timer},
		CandidateSession: accessquery.CandidateSession{Current: candidateSessions, Candidates: candidates, Assignments: store, CSRF: csrf, Clock: timer},
		CSRF:             csrf,
		Sessions:         sessions,
		Origin:           cfg.AccessHTTPOrigin,
		SecureCookie:     strings.HasPrefix(cfg.AccessHTTPOrigin, "https://"),
		Limiter:          &accesshandler.LoginLimiter{},
		ExchangeLimiter:  &accesshandler.LoginLimiter{},
		LogError:         logErr,
	}
}
