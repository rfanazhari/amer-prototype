package candidateinterviewhandler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

const executionsPath = "/candidate/executions"

// intentKeyPattern validates the transport key before session or CSRF lookup.
var intentKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// startRequest permits only the stored consent identity.
type startRequest struct {
	ConsentID string `json:"consentId"`
}

// takeoverRequest carries the new tab and the observed controller version.
type takeoverRequest struct {
	ControllerID    string `json:"controllerId"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

// cancelRequest carries the observed execution version.
type cancelRequest struct {
	ExpectedVersion int64 `json:"expectedVersion"`
}

// executionRoute returns the execution id and action for the four published execution routes.
func executionRoute(path string) (string, string, bool) {
	if path == executionsPath {
		return "", "start", true
	}
	if !strings.HasPrefix(path, executionsPath+"/") {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(path, executionsPath+"/"), "/")
	if parts[0] == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return parts[0], "poll", true
	}
	if len(parts) == 2 && (parts[1] == "takeover" || parts[1] == "cancel") {
		return parts[0], parts[1], true
	}
	return "", "", false
}

// execution dispatches a published action after method, origin and transport validation.
func (h Handler) execution(w http.ResponseWriter, r *http.Request, id, action string) {
	method := http.MethodPost
	if action == "poll" {
		method = http.MethodGet
	}
	if r.Method != method {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	if err := requireOrigin(r, h.Origin); err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	if action == "poll" {
		h.poll(w, r, id)
		return
	}
	if err := requireJSONContentType(r); err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	switch action {
	case "start":
		h.start(w, r)
	case "takeover":
		h.takeover(w, r, id)
	case "cancel":
		h.cancel(w, r, id)
	}
}

// poll reads an execution scoped by the operation to the authenticated candidate assignment.
func (h Handler) poll(w http.ResponseWriter, r *http.Request, id string) {
	r, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	v, err := h.GetExecution.Execute(r.Context(), candidateinterviewquery.GetExecutionInput{ExecutionID: id, ControllerID: r.Header.Get(controllerHeader)})
	if err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: executionView(*v)})
}

// start checks the repeatable intent before asking the operation to commit or find the execution.
func (h Handler) start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if readJSON(r, &req) != nil || !intentKeyPattern.MatchString(r.Header.Get("Idempotency-Key")) || uuid.Validate(req.ConsentID) != nil || !validTab(r) {
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.Start.Execute(r.Context(), candidateinterviewusecase.StartInterviewInput{ConsentID: req.ConsentID, IdempotencyKey: r.Header.Get("Idempotency-Key"), ControllerID: r.Header.Get(controllerHeader)})
	if err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	respond(w, status, envelope{Status: "success", Data: executionView(result.Execution)})
}

// takeover requires the body tab to match the header before session and CSRF checks.
func (h Handler) takeover(w http.ResponseWriter, r *http.Request, id string) {
	var req takeoverRequest
	if readJSON(r, &req) != nil || req.ExpectedVersion < 1 || !validTab(r) || req.ControllerID != r.Header.Get(controllerHeader) {
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.Takeover.Execute(r.Context(), candidateinterviewusecase.TakeoverInput{ExecutionID: id, ControllerID: req.ControllerID, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: takeoverBody{IsCurrentController: result.IsCurrentController, ControllerVersion: result.ControllerVersion}})
}

// takeoverBody exposes only controller ownership and its version.
type takeoverBody struct {
	IsCurrentController bool  `json:"isCurrentController"`
	ControllerVersion   int64 `json:"controllerVersion"`
}

// cancel validates the observed execution version before asking lifecycle for an early stop.
func (h Handler) cancel(w http.ResponseWriter, r *http.Request, id string) {
	var req cancelRequest
	if readJSON(r, &req) != nil || req.ExpectedVersion < 1 || !validTab(r) {
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
		return
	}
	r, ok := h.verifyCSRF(w, r)
	if !ok {
		return
	}
	result, err := h.Cancel.Execute(r.Context(), candidateinterviewusecase.CancelInterviewInput{ExecutionID: id, ControllerID: r.Header.Get(controllerHeader), ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		h.sendError(w, r, err, errs.CINDependencyUnavailable)
		return
	}
	respond(w, http.StatusOK, envelope{Status: "success", Data: executionView(*result)})
}

// validTab checks the tab identifier without duplicating domain validity rules.
func validTab(r *http.Request) bool {
	_, err := candidateinterview.ParseControllerID(r.Header.Get(controllerHeader))
	return err == nil
}

// executionView maps detached facts to the candidate-safe transport allow-list.
func executionView(v dto.InterviewExecutionView) executionBody {
	body := executionBody{ID: v.ID, State: v.State, Version: v.Version, IsCurrentController: v.IsCurrentController, ControllerVersion: v.ControllerVersion}
	if v.HardDeadline != nil {
		deadline := formatted(*v.HardDeadline)
		body.HardDeadline = &deadline
	}
	return body
}
