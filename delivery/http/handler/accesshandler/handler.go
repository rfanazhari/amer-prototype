package accesshandler

import (
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
)

// LoginLimiter bounds administrative login attempts by canonical email and client IP.
type LoginLimiter struct{}

// Handler translates administrative HTTP requests through injected access operations.
type Handler struct {
	Login        accessusecase.LoginOperation
	Current      accessquery.CurrentSessionOperation
	Provision    accessusecase.ProvisionHROperation
	Logout       accessusecase.LogoutOperation
	CSRF         service.CSRFTokens
	Sessions     service.CurrentAdministrativeSession
	Origin       string
	SecureCookie bool
	Limiter      *LoginLimiter
	LogError     func(error)
}

// ServeHTTP dispatches the four administrative access paths.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
