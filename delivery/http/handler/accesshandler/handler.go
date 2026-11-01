package accesshandler

import (
	"net/http"
	"strconv"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentbindingusecase"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// Handler translates administrative HTTP requests through injected access operations.
type Handler struct {
	Login            accessusecase.LoginOperation
	Current          accessquery.CurrentSessionOperation
	Provision        accessusecase.ProvisionHROperation
	Logout           accessusecase.LogoutOperation
	CreateCandidate  accessusecase.CreateCandidateOperation
	CreateAssignment agentbindingusecase.CreateAssignmentOperation
	Candidate        accessquery.CandidateOperation
	IssueInvitation  accessusecase.IssueInvitationOperation
	Exchange         accessusecase.ExchangeInvitationOperation
	CandidateSession accessquery.CandidateSessionOperation
	CSRF             service.CSRFTokens
	Sessions         service.CurrentAdministrativeSession
	Origin           string
	SecureCookie     bool
	Limiter          *LoginLimiter
	ExchangeLimiter  *LoginLimiter
	LogError         func(error)
}

// ServeHTTP dispatches the four administrative access paths.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/admin/sessions":
		h.login(w, r)
	case "/admin/session":
		h.current(w, r)
	case "/admin/sessions/current":
		h.logout(w, r)
	case "/admin/hr-accounts":
		h.provision(w, r)
	default:
		if h.dispatchCandidateRoutes(w, r) {
			return
		}
		fail(w, r, http.StatusNotFound, errs.NotFound)
	}
}

// login establishes an administrative session cookie after origin, rate and commit checks.
func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return
	}
	if err := requireJSONContentType(r, true); err != nil {
		h.sendError(w, r, err)
		return
	}
	var req loginRequest
	if err := readJSON(r, &req, false); err != nil {
		h.sendError(w, r, err)
		return
	}
	email, _ := service.CanonicalAdministrativeEmail(req.Email)
	if ok, retry := h.Limiter.allow(email, clientIP(r.RemoteAddr)); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		fail(w, r, http.StatusTooManyRequests, errs.IARateLimited)
		return
	}
	result, err := h.Login.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	setSessionCookie(w, h.SecureCookie, result.SessionToken, result.Session.ExpiresAt)
	respond(w, http.StatusCreated, envelope{Status: "success", Data: loginBody{
		Account: accountBody{ID: result.Account.ID, Name: result.Account.Name, Email: result.Account.Email, Role: result.Account.Role},
		Session: sessionBody{ID: result.Session.ID, ExpiresAt: formatted(result.Session.ExpiresAt)},
	}})
}

// current returns the bound administrative session and CSRF material.
func (h Handler) current(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	result, err := h.Current.Execute(r.Context())
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: currentView(result)})
}

// provision creates an HR account under the current ADMIN after CSRF verification.
func (h Handler) provision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return
	}
	if err := requireJSONContentType(r, true); err != nil {
		h.sendError(w, r, err)
		return
	}
	var req provisionRequest
	if err := readJSON(r, &req, false); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.Provision.Execute(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, envelope{Status: "success", Data: provisionBody{
		ID: result.ID, Name: result.Name, Email: result.Email, Role: result.Role, Status: result.Status,
		CreatedAt: formatted(result.CreatedAt), UpdatedAt: formatted(result.UpdatedAt),
	}})
}

// logout revokes the current session then clears the local cookie, including commit uncertainty.
func (h Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return
	}
	if err := requireJSONContentType(r, r.ContentLength > 0); err != nil {
		h.sendError(w, r, err)
		return
	}
	var body emptyRequest
	if err := readJSON(r, &body, true); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	err := h.Logout.Execute(r.Context())
	clearSessionCookie(w, h.SecureCookie)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: map[string]any{}})
}

// currentView copies the detached current-session DTO into the transport payload.
func currentView(result *dto.AdministrativeCurrentSession) currentBody {
	return currentBody{
		Account: accountBody{
			ID: result.Account.ID, Name: result.Account.Name, Email: result.Account.Email,
			Role: result.Account.Role, Status: result.Account.Status,
		},
		Session:   sessionBody{ID: result.Session.ID, ExpiresAt: formatted(result.Session.ExpiresAt)},
		CSRFToken: result.CSRFToken,
	}
}
