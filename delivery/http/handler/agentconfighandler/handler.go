package agentconfighandler

import (
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/query/agentconfigquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// Handler translates configuration HTTP requests through injected application operation interfaces.
type Handler struct {
	Catalog      agentconfigquery.CatalogOperation
	Create       agentconfigusecase.CreateOperation
	Instance     agentconfigquery.InstanceOperation
	Replace      agentconfigusecase.ReplaceOperation
	Validate     agentconfigusecase.ValidateOperation
	Publish      agentconfigusecase.PublishOperation
	Revision     agentconfigquery.RevisionOperation
	CSRF         service.CSRFTokens
	Sessions     service.CurrentAdministrativeSession
	Origin       string
	SecureCookie bool
	LogError     func(error)
}

// ServeHTTP dispatches the seven administrative configuration routes.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fail(w, r, http.StatusNotFound, errs.NotFound)
}
