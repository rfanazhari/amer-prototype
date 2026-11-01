package candidateinterviewhandler

import (
	"errors"
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

const (
	interviewPath    = "/candidate/interview"
	consentsPath     = "/candidate/consents"
	controllerHeader = "X-Interview-Controller"
)

// Handler translates candidate interview HTTP requests through injected application operation interfaces.
type Handler struct {
	GetExecution candidateinterviewquery.GetExecutionOperation
	Start        candidateinterviewusecase.StartInterviewOperation
	Takeover     candidateinterviewusecase.TakeoverOperation
	Cancel       candidateinterviewusecase.CancelInterviewOperation
	Get          candidateinterviewquery.GetInterviewOperation
	Accept       candidateinterviewusecase.AcceptNoticeOperation
	Current      service.CurrentCandidateSession
	CSRF         service.CSRFTokens
	Origin       string
	SecureCookie bool
	LogError     func(error)
}

// noticeBody is the notice the candidate is asked to accept.
type noticeBody struct {
	Version   string `json:"version"`
	Statement string `json:"statement"`
}

// consentBody is one stored acceptance.
type consentBody struct {
	ID            string `json:"id"`
	NoticeVersion string `json:"noticeVersion"`
	AcceptedAt    string `json:"acceptedAt"`
}

// executionBody is the candidate-safe execution; it has no controller id, report or salary member.
type executionBody struct {
	ID                  string  `json:"id"`
	State               string  `json:"state"`
	Version             int64   `json:"version"`
	HardDeadline        *string `json:"hardDeadline"`
	IsCurrentController bool    `json:"isCurrentController"`
	ControllerVersion   int64   `json:"controllerVersion"`
}

// progressBody is the progress summary runtime marks as visible to the candidate.
type progressBody struct {
	Phase      string `json:"phase"`
	Connection string `json:"connection"`
}

// interviewBody is the screen model; absent members are JSON null and actions is never null.
type interviewBody struct {
	Notice    *noticeBody    `json:"notice"`
	Consent   *consentBody   `json:"consent"`
	Execution *executionBody `json:"execution"`
	Actions   []string       `json:"actions"`
	Progress  *progressBody  `json:"progress"`
}

// consentRequest accepts the notice version and the explicit acceptance only.
type consentRequest struct {
	NoticeVersion string `json:"noticeVersion"`
	Accepted      bool   `json:"accepted"`
}

// ServeHTTP dispatches the mounted candidate interview JSON paths.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case interviewPath:
		h.read(w, r)
	case consentsPath:
		h.accept(w, r)
	default:
		if id, action, ok := executionRoute(r.URL.Path); ok {
			h.execution(w, r, id, action)
			return
		}
		fail(w, r, http.StatusNotFound, errs.NotFound)
	}
}

// read returns the screen model for the bound candidate session without starting an interview.
func (h Handler) read(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	r, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	view, err := h.Get.Execute(r.Context(), candidateinterviewquery.GetInterviewInput{ControllerID: r.Header.Get(controllerHeader)})
	if err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: interviewView(view)})
}

// accept records acceptance of the current notice version after strict input, session and CSRF checks.
func (h Handler) accept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err, errs.CINNoticeUnavailable)
		return
	}
	if err := requireJSONContentType(r); err != nil {
		h.sendError(w, r, err, errs.CINNoticeUnavailable)
		return
	}
	var req consentRequest
	if err := readJSON(r, &req); err != nil {
		h.sendError(w, r, err, errs.CINNoticeUnavailable)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.Accept.Execute(r.Context(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: req.NoticeVersion, Accepted: req.Accepted})
	if err != nil {
		h.sendError(w, r, err, errs.CINNoticeUnavailable)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	respond(w, status, envelope{Status: "success", Data: consentView(result.Consent)})
}

// authenticate binds the candidate cookie or rejects administrative substitution and a missing cookie.
func (h Handler) authenticate(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	token, ok, admin := accesshandler.ReadCandidateCookie(r, h.SecureCookie)
	if admin || !ok {
		fail(w, r, http.StatusUnauthorized, errs.IAUnauthenticated)
		return r, false
	}
	return accesshandler.BindCandidateCredential(r, token), true
}

// verifyCSRF authenticates, resolves the session and accepts only a session-bound CSRF header.
func (h Handler) verifyCSRF(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	r, ok := h.authenticate(w, r)
	if !ok {
		return r, false
	}
	session, err := h.Current.Resolve(r.Context())
	if err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return r, false
	}
	token := r.Header.Get("X-CSRF-Token")
	if token == "" || h.CSRF.Verify(service.CredentialCandidate, session.ID, token) != nil {
		fail(w, r, http.StatusForbidden, errs.CINForbidden)
		return r, false
	}
	return r, true
}

// sendError maps classified failures onto safe localized envelopes; dependencyCode names the route's 503 code.
func (h Handler) sendError(w http.ResponseWriter, r *http.Request, err error, dependencyCode string) {
	var technical errs.Technical
	if errors.As(err, &technical) {
		h.log(err)
		fail(w, r, http.StatusInternalServerError, errs.InternalError)
		return
	}
	switch {
	// Input validation returns these sentinels directly. Storage restoration wraps them;
	// those failures must reach the technical fallback instead of becoming client errors.
	case errors.Is(err, errInput), err == candidateinterview.ErrInvalidConsent,
		err == candidateinterview.ErrInvalidNoticeVersion, err == candidateinterview.ErrNoticeNotCurrent,
		err == candidateinterview.ErrInvalidControllerID, errors.Is(err, candidateinterviewusecase.ErrInvalidIdempotencyKey):
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
	case errors.Is(err, service.ErrCandidateSessionUnbound):
		fail(w, r, http.StatusUnauthorized, errs.IAUnauthenticated)
	case errors.Is(err, service.ErrInterviewNotFound):
		fail(w, r, http.StatusNotFound, errs.NotFound)
	case errors.Is(err, candidateinterviewusecase.ErrConsentRequired):
		fail(w, r, http.StatusConflict, errs.CINConsentRequired)
	case errors.Is(err, service.ErrInterviewCoverage):
		fail(w, r, http.StatusForbidden, errs.CINSessionCoverage)
	case errors.Is(err, service.ErrInterviewConflict), errors.Is(err, candidateinterview.ErrStaleControllerVersion):
		code := errs.CINControllerConflict
		if r.URL.Path == executionsPath {
			code = errs.CINIdempotencyConflict
		}
		fail(w, r, http.StatusConflict, code)
	case errors.Is(err, candidateinterviewusecase.ErrNotController):
		fail(w, r, http.StatusForbidden, errs.CINNotController)
	case errors.Is(err, candidateinterview.ErrExecutionTerminal):
		fail(w, r, http.StatusConflict, errs.CINTerminal)
	case errors.Is(err, service.ErrInterviewNotLive):
		fail(w, r, http.StatusConflict, errs.CINNotLive)
	case errors.Is(err, service.ErrInterviewStartUncertain):
		fail(w, r, http.StatusServiceUnavailable, errs.CINStartUncertain)
	case errors.Is(err, service.ErrInterviewDependencyUnavailable):
		fail(w, r, http.StatusServiceUnavailable, dependencyCode)
	default:
		h.log(err)
		fail(w, r, http.StatusInternalServerError, errs.InternalError)
	}
}

// log reports an unexpected failure to the injected sink without exposing it to the caller.
func (h Handler) log(err error) {
	if h.LogError != nil {
		h.LogError(err)
	}
}

// consentView copies the detached consent DTO into the transport payload.
func consentView(c dto.ConsentView) consentBody {
	return consentBody{ID: c.ID, NoticeVersion: c.NoticeVersion, AcceptedAt: formatted(c.AcceptedAt)}
}

// interviewView copies the detached screen DTO into the allow-list transport payload.
func interviewView(v *dto.InterviewView) interviewBody {
	body := interviewBody{Actions: append([]string{}, v.Actions...)}
	if v.Notice != nil {
		body.Notice = &noticeBody{Version: v.Notice.Version, Statement: v.Notice.Statement}
	}
	if v.Consent != nil {
		c := consentView(*v.Consent)
		body.Consent = &c
	}
	if v.Execution != nil {
		e := executionView(*v.Execution)
		body.Execution = &e
	}
	if v.Progress != nil {
		body.Progress = &progressBody{Phase: v.Progress.Phase, Connection: v.Progress.Connection}
	}
	return body
}
