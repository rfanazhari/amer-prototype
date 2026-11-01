package candidateinterviewhandler

import (
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
)

// Handler translates candidate interview HTTP requests through injected application operation interfaces.
type Handler struct {
	Get          candidateinterviewquery.GetInterviewOperation
	Accept       candidateinterviewusecase.AcceptNoticeOperation
	Current      service.CurrentCandidateSession
	CSRF         service.CSRFTokens
	Origin       string
	SecureCookie bool
	LogError     func(error)
}

// ServeHTTP is not implemented yet.
func (Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
