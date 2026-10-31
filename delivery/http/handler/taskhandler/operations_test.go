package taskhandler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/taskhandler"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOperationInterfacesAndEnvelopes(t *testing.T) {
	result := dto.FromTask(testdata.Task(t, enum.Todo, nil), testdata.Now())
	create := mocks.NewCreateOperation(t)
	start := mocks.NewStartOperation(t)
	complete := mocks.NewCompleteOperation(t)
	reschedule := mocks.NewRescheduleOperation(t)
	detail := mocks.NewDetailOperation(t)
	create.On("Execute", mock.Anything, mock.Anything, "task", mock.Anything).Return(result, nil)
	start.On("Execute", mock.Anything, testdata.ID).Return(result, nil)
	complete.On("Execute", mock.Anything, testdata.ID).Return(result, nil)
	reschedule.On("Execute", mock.Anything, testdata.ID, mock.Anything).Return(result, nil)
	detail.On("Execute", mock.Anything, testdata.ID).Return(result, nil)
	handler := taskhandler.Handler{Create: create, Start: start, Complete: complete, Reschedule: reschedule, Detail: detail}
	type requestCase struct {
		name, method, path, body string
		status                   int
	}
	tests := []requestCase{{"create", "POST", "/tasks", `{"title":"task"}`, 201}, {"start", "POST", "/tasks/" + testdata.ID + "/start", "", 200}, {"complete", "POST", "/tasks/" + testdata.ID + "/complete", "{}", 200}, {"clear due date", "PATCH", "/tasks/" + testdata.ID + "/due-date", `{"dueDate":null}`, 200}, {"detail", "GET", "/tasks/" + testdata.ID, "", 200}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			require.Equal(t, tt.status, w.Code)
			require.Contains(t, w.Body.String(), `"status":"success"`)
			require.Contains(t, w.Body.String(), `"dueDate":null`)
			require.Contains(t, w.Body.String(), `"completedAt":null`)
		})
	}
}

type errorCase struct {
	name   string
	cause  error
	status int
	code   string
	logged bool
}

func TestErrorClassification(t *testing.T) {
	tests := []errorCase{{"not found", repository.ErrNotFound, 404, errs.NotFound, false}, {"transition", aggregate.ErrTransition, 409, errs.InvalidTransition, false}, {"queue refused", service.ErrEventUnavailable, 503, errs.EventUnavailable, false}, {"invalid title", valueobject.ErrInvalidTitle, 400, errs.InvalidInput, false}, {"corrupt persisted title", errs.Technical{Cause: valueobject.ErrInvalidTitle}, 500, errs.InternalError, true}, {"database failure", errors.New("secret database diagnostic"), 500, errs.InternalError, true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := mocks.NewStartOperation(t)
			operation.On("Execute", mock.Anything, testdata.ID).Return(nil, tt.cause)
			logged := false
			handler := taskhandler.Handler{Start: operation, LogError: func(err error) { logged = true; require.ErrorIs(t, err, tt.cause) }}
			request := httptest.NewRequest(http.MethodPost, "/tasks/"+testdata.ID+"/start", nil)
			request.Header.Set("Accept-Language", "id")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, request)
			require.Equal(t, tt.status, w.Code)
			require.Contains(t, w.Body.String(), `"code":"`+tt.code+`"`)
			require.Contains(t, w.Body.String(), `"data":{}`)
			require.NotContains(t, w.Body.String(), "secret database diagnostic")
			require.Equal(t, tt.logged, logged)
		})
	}
}
