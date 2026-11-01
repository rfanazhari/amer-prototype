package accesshandler

import (
	"errors"
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// sendError maps classified access failures onto safe localized HTTP envelopes.
func (h Handler) sendError(w http.ResponseWriter, r *http.Request, err error) {
	var technical errs.Technical
	if errors.As(err, &technical) {
		if h.LogError != nil {
			h.LogError(err)
		}
		fail(w, r, http.StatusInternalServerError, errs.InternalError)
		return
	}
	var conflict accessport.AccountConflictError
	var uncertain *accessport.CommitUncertainError
	switch {
	case errors.Is(err, errInput), errors.Is(err, service.ErrInvalidInput):
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
	case errors.Is(err, service.ErrAuthenticationFailed), errors.Is(err, service.ErrSessionUnbound), errors.Is(err, service.ErrCandidateSessionUnbound):
		fail(w, r, http.StatusUnauthorized, errs.IAUnauthenticated)
	case errors.Is(err, service.ErrCSRFMismatch), errors.Is(err, service.ErrForbidden):
		fail(w, r, http.StatusForbidden, errs.IAForbidden)
	case errors.Is(err, accessport.ErrNotFound):
		fail(w, r, http.StatusNotFound, errs.NotFound)
	case errors.Is(err, accessport.ErrInvitationUnavailable):
		fail(w, r, http.StatusGone, errs.IAInvitationUnavailable)
	case errors.As(err, &conflict):
		fail(w, r, http.StatusConflict, errs.IAAccountConflict)
	case errors.As(err, &uncertain):
		fail(w, r, http.StatusServiceUnavailable, errs.IADependencyUnavailable)
	default:
		if h.LogError != nil {
			h.LogError(err)
		}
		fail(w, r, http.StatusInternalServerError, errs.InternalError)
	}
}

// authenticate binds the administrative cookie or rejects candidate substitution and missing sessions.
func (h Handler) authenticate(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	token, ok, candidate := readAdminCookie(r, h.SecureCookie)
	if candidate || !ok {
		fail(w, r, http.StatusUnauthorized, errs.IAUnauthenticated)
		return r, false
	}
	return bindCredential(r, token), true
}

// authenticateCandidate binds the candidate cookie or rejects administrative substitution and missing sessions.
func (h Handler) authenticateCandidate(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	token, ok, admin := readCandidateCookie(r, h.SecureCookie)
	if admin || !ok {
		fail(w, r, http.StatusUnauthorized, errs.IAUnauthenticated)
		return r, false
	}
	return bindCandidateCredential(r, token), true
}

// verifyCSRF authenticates then accepts a session-bound CSRF header on unsafe methods.
func (h Handler) verifyCSRF(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	r, ok := h.authenticate(w, r)
	if !ok {
		return r, false
	}
	session, _, err := h.Sessions.Resolve(r.Context())
	if err != nil {
		h.sendError(w, r, err)
		return r, false
	}
	token := r.Header.Get("X-CSRF-Token")
	if token == "" || h.CSRF.Verify(service.CredentialAdministrative, session.ID, token) != nil {
		fail(w, r, http.StatusForbidden, errs.IAForbidden)
		return r, false
	}
	return r, true
}
