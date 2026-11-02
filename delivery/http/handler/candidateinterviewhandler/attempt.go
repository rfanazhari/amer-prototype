package candidateinterviewhandler

import (
	"net/http"
	"strings"

	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// attemptRequest permits only the five published session-attempt members.
type attemptRequest struct {
	SDP             string `json:"sdp"`
	Intent          string `json:"intent"`
	ExpectedVersion int64  `json:"expectedVersion"`
	ControllerID    string `json:"controllerId"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

// readyRequest carries the tab and the observed execution version.
type readyRequest struct {
	ControllerID    string `json:"controllerId"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

// attemptBody is the session-attempt result; sdp is the provider answer only.
type attemptBody struct {
	AttemptID        string `json:"attemptId"`
	SessionID        string `json:"sessionId"`
	SDP              string `json:"sdp"`
	ExecutionVersion int64  `json:"executionVersion"`
	SessionCondition string `json:"sessionCondition"`
}

// readyBody is the readiness result; it carries no provider data.
type readyBody struct {
	AttemptID        string `json:"attemptId"`
	SessionID        string `json:"sessionId"`
	ExecutionVersion int64  `json:"executionVersion"`
	SessionCondition string `json:"sessionCondition"`
}

// attemptRoute matches POST .../session-attempts and POST .../session-attempts/{attempt_id}/ready.
func attemptRoute(path string) (executionID, attemptID, action string, ok bool) {
	rest, found := strings.CutPrefix(path, executionsPath+"/")
	if !found {
		return "", "", "", false
	}
	parts := strings.Split(rest, "/")
	switch {
	case len(parts) == 2 && parts[0] != "" && parts[1] == "session-attempts":
		return parts[0], "", "attempt", true
	case len(parts) == 4 && parts[0] != "" && parts[1] == "session-attempts" && parts[2] != "" && parts[3] == "ready":
		return parts[0], parts[2], "ready", true
	}
	return "", "", "", false
}

// sessionAttempt applies method, origin, content type and strict body validation before the session and CSRF checks.
func (h Handler) sessionAttempt(w http.ResponseWriter, r *http.Request, executionID, attemptID, action string) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	if err := requireJSONContentType(r); err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	if action == "ready" {
		h.ready(w, r, executionID, attemptID)
		return
	}
	h.attempt(w, r, executionID)
}

// attempt creates or replays a voice attempt; 201 means a new attempt and 200 the original result.
func (h Handler) attempt(w http.ResponseWriter, r *http.Request, executionID string) {
	var req attemptRequest
	if readJSON(r, &req) != nil || strings.TrimSpace(req.SDP) == "" || (req.Intent != interviewusecase.IntentActivate && req.Intent != interviewusecase.IntentReconnect) ||
		req.ExpectedVersion < 1 || !intentKeyPattern.MatchString(req.IdempotencyKey) || !validControllerID(req.ControllerID) {
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.Attempt.Execute(r.Context(), interviewusecase.AttemptInput{
		ExecutionID: executionID, ControllerID: req.ControllerID, IdempotencyKey: req.IdempotencyKey,
		Intent: req.Intent, SDP: req.SDP, ExpectedVersion: req.ExpectedVersion,
	})
	if err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	respond(w, status, envelope{Status: "success", Data: attemptBody{
		AttemptID: result.AttemptID, SessionID: result.SessionID, SDP: result.SDP,
		ExecutionVersion: result.ExecutionVersion, SessionCondition: result.SessionCondition,
	}})
}

// ready records browser readiness for one attempt; it never marks a question heard.
func (h Handler) ready(w http.ResponseWriter, r *http.Request, executionID, attemptID string) {
	var req readyRequest
	if readJSON(r, &req) != nil || req.ExpectedVersion < 1 || !validControllerID(req.ControllerID) {
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.Ready.Execute(r.Context(), interviewusecase.ReadyInput{ExecutionID: executionID, AttemptID: attemptID, ControllerID: req.ControllerID, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: readyBody{
		AttemptID: result.AttemptID, SessionID: result.SessionID, ExecutionVersion: result.ExecutionVersion, SessionCondition: result.SessionCondition,
	}})
}

// validControllerID checks a body tab identifier without duplicating domain validity rules.
func validControllerID(raw string) bool {
	_, err := candidateinterview.ParseControllerID(raw)
	return err == nil
}
