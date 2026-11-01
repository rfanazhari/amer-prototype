package accesshandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

const (
	candidatesPath       = "/admin/candidates"
	invitationsPath      = "/invitations"
	assignmentsPath      = "/admin/interview-assignments"
	assignmentsRoot      = "/admin/interview-assignments/"
	exchangePath         = "/candidate/access/exchange"
	candidateSessionPath = "/candidate/session"
)

// dispatchCandidateRoutes handles candidate identity, invitation and session HTTP paths.
func (h Handler) dispatchCandidateRoutes(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case r.URL.Path == candidatesPath:
		h.createCandidate(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, candidatesPath+"/"):
		h.readCandidate(w, r)
		return true
	case r.URL.Path == assignmentsPath:
		h.createAssignment(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, assignmentsRoot) && strings.HasSuffix(r.URL.Path, invitationsPath):
		h.issueInvitation(w, r)
		return true
	case r.URL.Path == exchangePath:
		h.exchange(w, r)
		return true
	case r.URL.Path == candidateSessionPath:
		h.candidateCurrent(w, r)
		return true
	default:
		return false
	}
}

// createCandidate records a reusable candidate identity under current HR authority.
func (h Handler) createCandidate(w http.ResponseWriter, r *http.Request) {
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
	var req candidateRequest
	if err := readJSON(r, &req, false); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.CreateCandidate.Execute(r.Context(), req.Name, req.Email)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, envelope{Status: "success", Data: candidateBody{
		ID: result.ID, Name: result.Name, Email: result.Email,
		CreatedAt: formatted(result.CreatedAt), UpdatedAt: formatted(result.UpdatedAt),
	}})
}

// readCandidate returns one candidate after current HR authorization.
func (h Handler) readCandidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	id, ok := pathTail(r.URL.Path, candidatesPath+"/")
	if !ok {
		fail(w, r, http.StatusNotFound, errs.NotFound)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok = h.authenticate(w, r)
	if !ok {
		return
	}
	result, err := h.Candidate.Execute(r.Context(), id)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: candidateBody{
		ID: result.ID, Name: result.Name, Email: result.Email,
		CreatedAt: formatted(result.CreatedAt), UpdatedAt: formatted(result.UpdatedAt),
	}})
}

// createAssignment pins an existing candidate to the current HR Interviewer active revision.
func (h Handler) createAssignment(w http.ResponseWriter, r *http.Request) {
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
	var req assignmentRequest
	if err := readJSON(r, &req, false); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.CreateAssignment.Execute(r.Context(), req.CandidateID, req.InstanceID)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, envelope{Status: "success", Data: assignmentBody{
		ID: result.ID, CandidateID: result.CandidateID, InstanceID: result.InstanceID, PublishedRevisionID: result.PublishedRevisionID,
	}})
}

// issueInvitation issues or replaces an assignment invitation under current HR authority.
func (h Handler) issueInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	assignmentID, ok := assignmentInvitationID(r.URL.Path)
	if !ok {
		fail(w, r, http.StatusNotFound, errs.NotFound)
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
	r, ok = h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.IssueInvitation.Execute(r.Context(), assignmentID)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, envelope{Status: "success", Data: invitationBody{
		ID: result.ID, AssignmentID: result.AssignmentID, ExpiresAt: formatted(result.ExpiresAt), Token: result.Token,
	}})
}

// exchange establishes a candidate session cookie after origin, rate and commit checks.
func (h Handler) exchange(w http.ResponseWriter, r *http.Request) {
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
	var req exchangeRequest
	if err := readJSON(r, &req, false); err != nil {
		h.sendError(w, r, err)
		return
	}
	if ok, retry := h.ExchangeLimiter.allow(req.Token, clientIP(r.RemoteAddr)); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		fail(w, r, http.StatusTooManyRequests, errs.IARateLimited)
		return
	}
	result, err := h.Exchange.Execute(r.Context(), req.Token)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	setCandidateCookie(w, h.SecureCookie, result.SessionToken, result.Session.ExpiresAt)
	respond(w, http.StatusCreated, envelope{Status: "success", Data: candidateAccessView(result, false)})
}

// candidateCurrent returns the bound candidate session and CSRF material without exchanging.
func (h Handler) candidateCurrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err)
		return
	}
	r, ok := h.authenticateCandidate(w, r)
	if !ok {
		return
	}
	result, err := h.CandidateSession.Execute(r.Context())
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: candidateAccessView(result, true)})
}

// candidateAccessView copies the detached candidate access DTO into the transport payload.
func candidateAccessView(result *dto.CandidateAccess, includeCSRF bool) candidateAccessBody {
	view := candidateAccessBody{
		Candidate:  candidateIdentityBody{ID: result.Candidate.ID, Name: result.Candidate.Name},
		Assignment: assignmentRefBody{ID: result.AssignmentID},
		Session:    sessionBody{ID: result.Session.ID, ExpiresAt: formatted(result.Session.ExpiresAt)},
	}
	if includeCSRF {
		view.CSRFToken = result.CSRFToken
	}
	return view
}

// pathTail returns the single remaining path segment after prefix.
func pathTail(path, prefix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(path, prefix)
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// assignmentInvitationID extracts the assignment identity from the invitation issue path.
func assignmentInvitationID(path string) (string, bool) {
	if !strings.HasPrefix(path, assignmentsRoot) || !strings.HasSuffix(path, invitationsPath) {
		return "", false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, assignmentsRoot), invitationsPath)
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}
