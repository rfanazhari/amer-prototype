package agentconfighandler

import (
	"errors"
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// sendError maps classified configuration failures onto safe localized HTTP envelopes.
func (h Handler) sendError(w http.ResponseWriter, r *http.Request, err error) {
	var technical errs.Technical
	if errors.As(err, &technical) {
		if h.LogError != nil {
			h.LogError(err)
		}
		fail(w, r, http.StatusInternalServerError, errs.InternalError)
		return
	}
	var uncertain *agentconfigport.CommitUncertainError
	switch {
	case errors.Is(err, errInput), errors.Is(err, service.ErrInvalidInput), errors.Is(err, catalog.ErrUnknownTemplate), errors.Is(err, valueobject.ErrInvalidConfiguration):
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
	case errors.Is(err, service.ErrSessionUnbound), errors.Is(err, service.ErrCandidateSessionUnbound), errors.Is(err, service.ErrAuthenticationFailed):
		fail(w, r, http.StatusUnauthorized, errs.ACPUnauthenticated)
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrCSRFMismatch):
		fail(w, r, http.StatusForbidden, errs.ACPForbidden)
	case errors.Is(err, agentconfigport.ErrNotFound):
		fail(w, r, http.StatusNotFound, errs.NotFound)
	case errors.Is(err, service.ErrIdempotencyConflict):
		fail(w, r, http.StatusConflict, errs.ACPIdempotencyConflict)
	case errors.Is(err, service.ErrDraftConflict), errors.Is(err, aggregate.ErrStale):
		fail(w, r, http.StatusConflict, errs.ACPDraftConflict)
	case errors.Is(err, service.ErrValidationConflict):
		fail(w, r, http.StatusConflict, errs.ACPValidationConflict)
	case errors.Is(err, service.ErrPublicationBlocked):
		fail(w, r, http.StatusUnprocessableEntity, errs.ACPPublicationBlocked)
	case errors.Is(err, service.ErrDependencyUnavailable):
		fail(w, r, http.StatusServiceUnavailable, errs.ACPDependencyUnavailable)
	case errors.As(err, &uncertain):
		fail(w, r, http.StatusServiceUnavailable, errs.ACPPublicationUncertain)
	default:
		if h.LogError != nil {
			h.LogError(err)
		}
		fail(w, r, http.StatusInternalServerError, errs.InternalError)
	}
}
