// Package interviewreporthandler maps the HR report HTTP contract onto application operations.
package interviewreporthandler

import (
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/query/interviewreportquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
)

// Handler serves report reads and explicit regenerations without registering runtime routes.
type Handler struct {
	Read         interviewreportquery.ReadOperation
	Regenerate   interviewreportusecase.RegenerateOperation
	Sessions     service.CurrentAdministrativeSession
	CSRF         service.CSRFTokens
	Origin       string
	SecureCookie bool
	LogError     func(error)
}

// ServeHTTP translates the report routes into application calls.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}
