package taskhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/taskquery"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/taskusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/translations"
)

var errInput = errors.New("invalid input")

// Handler translates Task HTTP requests and envelopes through injected application operation interfaces.
type Handler struct {
	Create     taskusecase.CreateOperation
	Start      taskusecase.StartOperation
	Complete   taskusecase.CompleteOperation
	Reschedule taskusecase.RescheduleOperation
	Detail     taskquery.DetailOperation
	LogError   func(error)
}

// envelope keeps status and data present on every response, with error included only on failures.
type envelope struct {
	Status string     `json:"status"`
	Data   any        `json:"data"`
	Error  *errorBody `json:"error,omitempty"`
}

// errorBody carries a stable public code and a localized message without technical causes.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// taskBody formats application results for JSON while retaining null optional timestamps.
type taskBody struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	DueDate     *string `json:"dueDate"`
	Overdue     bool    `json:"overdue"`
	CreatedAt   string  `json:"createdAt"`
	CompletedAt *string `json:"completedAt"`
}

// createRequest accepts client-owned creation fields; unknown and system-owned fields are rejected.
type createRequest struct {
	Title   string          `json:"title"`
	DueDate json.RawMessage `json:"dueDate"`
}

// emptyRequest permits an empty action object without accepting unexpected fields.
type emptyRequest struct{}

// dateRequest preserves the distinction between an omitted dueDate and an explicit null.
type dateRequest struct {
	DueDate json.RawMessage `json:"dueDate"`
}

// respond writes the standard JSON envelope and selected HTTP status.
func respond(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// fail localizes a stable error code and emits an empty data object in the error envelope.
func fail(w http.ResponseWriter, r *http.Request, status int, code string) {
	msg := translations.Message(r.Header.Get("Accept-Language"), code)
	respond(w, status, envelope{Status: "error", Data: map[string]any{}, Error: &errorBody{Code: code, Message: msg}})
}

// readJSON accepts one bounded JSON object, rejects unknown fields and trailing documents, and optionally permits an absent body.
func readJSON(r *http.Request, dst any, emptyAllowed bool) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1048577))
	if err != nil || len(body) > 1048576 {
		return errInput
	}
	if len(bytes.TrimSpace(body)) == 0 && emptyAllowed {
		return nil
	}
	if trimmed := bytes.TrimSpace(body); len(trimmed) == 0 || trimmed[0] != '{' {
		return errInput
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err = dec.Decode(dst); err != nil {
		return errInput
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return errInput
	}
	return nil
}

// parseDate accepts a null or RFC3339 instant with at most millisecond precision and normalizes valid offsets to UTC.
func parseDate(raw json.RawMessage) (*time.Time, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return nil, errInput
	}
	if strings.Contains(s, ",") {
		return nil, errInput
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		j := i + 1
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j-i-1 > 3 {
			return nil, errInput
		}
	}
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, errInput
	}
	v = v.UTC()
	return &v, nil
}

// formatted renders an optional UTC instant with exactly three fractional digits.
func formatted(v *time.Time) *string {
	if v == nil {
		return nil
	}
	s := v.UTC().Format("2006-01-02T15:04:05.000Z")
	return &s
}

// details maps the detached application result into transport timestamp strings and nullable fields.
func details(task *dto.Task) taskBody {
	created := formatted(&task.CreatedAt)
	return taskBody{ID: task.ID, Title: task.Title, Status: task.Status, DueDate: formatted(task.DueDate), Overdue: task.Overdue, CreatedAt: *created, CompletedAt: formatted(task.CompletedAt)}
}

// sendError maps typed failures to safe HTTP errors. Technical wrappers take precedence over their underlying business error causes.
func (h Handler) sendError(w http.ResponseWriter, r *http.Request, err error) {
	var technical errs.Technical
	if errors.As(err, &technical) {
		if h.LogError != nil {
			h.LogError(err)
		}
		fail(w, r, 500, errs.InternalError)
		return
	}
	switch {
	case errors.Is(err, errInput), errors.Is(err, valueobject.ErrInvalidTitle), errors.Is(err, aggregate.ErrDueDate):
		fail(w, r, 400, errs.InvalidInput)
	case errors.Is(err, repository.ErrNotFound):
		fail(w, r, 404, errs.NotFound)
	case errors.Is(err, aggregate.ErrTransition):
		fail(w, r, 409, errs.InvalidTransition)
	case errors.Is(err, service.ErrEventUnavailable):
		fail(w, r, 503, errs.EventUnavailable)
	default:
		if h.LogError != nil {
			h.LogError(err)
		}
		fail(w, r, 500, errs.InternalError)
	}
}

// ShuttingDown reports admission refusal using the standard localized 503 envelope.
func ShuttingDown(w http.ResponseWriter, r *http.Request) {
	fail(w, r, http.StatusServiceUnavailable, errs.ShuttingDown)
}

// ServeHTTP dispatches the Task and health routes, validates transport input and delegates business operations to application interfaces.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		if r.Method != http.MethodGet {
			fail(w, r, 405, errs.MethodNotAllowed)
			return
		}
		respond(w, 200, envelope{Status: "success", Data: map[string]any{}})
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) == 0 || parts[0] != "tasks" || len(parts) > 3 || (len(parts) == 3 && parts[2] != "start" && parts[2] != "complete" && parts[2] != "due-date") {
		fail(w, r, 404, errs.NotFound)
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodPost {
			fail(w, r, 405, errs.MethodNotAllowed)
			return
		}
		var req createRequest
		if err := readJSON(r, &req, false); err != nil {
			h.sendError(w, r, err)
			return
		}
		due, err := parseDate(req.DueDate)
		if err != nil {
			h.sendError(w, r, err)
			return
		}
		task, err := h.Create.Execute(r.Context(), uuid.NewString(), req.Title, due)
		if err != nil {
			h.sendError(w, r, err)
			return
		}
		respond(w, 201, envelope{Status: "success", Data: details(task)})
		return
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		h.sendError(w, r, errInput)
		return
	}
	var task *dto.Task
	var err error
	if len(parts) == 2 {
		if r.Method != http.MethodGet {
			fail(w, r, 405, errs.MethodNotAllowed)
			return
		}
		task, err = h.Detail.Execute(r.Context(), parts[1])
	} else {
		switch parts[2] {
		case "start", "complete":
			if r.Method != http.MethodPost {
				fail(w, r, 405, errs.MethodNotAllowed)
				return
			}
			var body emptyRequest
			if err = readJSON(r, &body, true); err != nil {
				h.sendError(w, r, err)
				return
			}
			if parts[2] == "start" {
				task, err = h.Start.Execute(r.Context(), parts[1])
			} else {
				task, err = h.Complete.Execute(r.Context(), parts[1])
			}
		case "due-date":
			if r.Method != http.MethodPatch {
				fail(w, r, 405, errs.MethodNotAllowed)
				return
			}
			var req dateRequest
			if err = readJSON(r, &req, false); err != nil || len(req.DueDate) == 0 {
				h.sendError(w, r, errInput)
				return
			}
			var due *time.Time
			due, err = parseDate(req.DueDate)
			if err == nil {
				task, err = h.Reschedule.Execute(r.Context(), parts[1], due)
			}
		}
	}
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, 200, envelope{Status: "success", Data: details(task)})
}
