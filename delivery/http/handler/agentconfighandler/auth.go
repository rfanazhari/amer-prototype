package agentconfighandler

import (
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// authenticate binds the administrative cookie or rejects candidate substitution and missing sessions.
func (h Handler) authenticate(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	token, ok, candidate := accesshandler.ReadAdminCookie(r, h.SecureCookie)
	if candidate || !ok {
		fail(w, r, http.StatusUnauthorized, errs.ACPUnauthenticated)
		return r, false
	}
	return accesshandler.BindAdministrativeCredential(r, token), true
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
		fail(w, r, http.StatusForbidden, errs.ACPForbidden)
		return r, false
	}
	return r, true
}

// prepareRead authenticates a safe method after origin verification.
func (h Handler) prepareRead(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return r, false
	}
	return h.authenticate(w, r)
}

// prepareMutation authenticates CSRF and JSON content type after origin verification.
func (h Handler) prepareMutation(w http.ResponseWriter, r *http.Request, hasBody bool) (*http.Request, bool) {
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return r, false
	}
	if err := requireJSONContentType(r, hasBody); err != nil {
		h.sendError(w, r, err)
		return r, false
	}
	return h.verifyCSRF(w, r)
}
