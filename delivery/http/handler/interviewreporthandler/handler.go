// Package interviewreporthandler maps the HR report HTTP contract onto application operations.
package interviewreporthandler

import (
	"net/http"
	"strings"

	"github.com/rfanazhari/amer-prototype/application/query/interviewreportquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
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
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 || parts[0] != "" || parts[1] != "admin" || parts[2] != "interview-assignments" || (parts[4] != "report" && parts[4] != "report-generations") {
		fail(w, r, http.StatusNotFound, errs.NotFound)
		return
	}
	post := parts[4] == "report-generations"
	if (!post && r.Method != http.MethodGet) || (post && r.Method != http.MethodPost) {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	r, ok := h.prepare(w, r, post)
	if !ok {
		return
	}
	id, err := parseInput(r, parts[3], post)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	if post {
		result, err := h.Regenerate.Execute(r.Context(), interviewreportusecase.RegenerateInput{AssignmentID: id, IdempotencyKey: r.Header.Get("Idempotency-Key")})
		if err != nil {
			h.sendError(w, r, err)
			return
		}
		status := http.StatusCreated
		if result.Replayed {
			status = http.StatusOK
		}
		respond(w, status, encodeGeneration(*result.Generation))
		return
	}
	view, err := h.Read.Execute(r.Context(), interviewreportquery.ReadInput{AssignmentID: id})
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, encodeView(*view))
}
