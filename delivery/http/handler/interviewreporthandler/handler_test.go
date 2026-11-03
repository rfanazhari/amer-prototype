package interviewreporthandler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/interviewreportquery"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// successCase describes read versus fresh and replayed regeneration responses.
type successCase struct {
	name    string
	post    bool
	replay  bool
	account bool
	secure  bool
	want    int
}

// TestReportSuccess verifies exact wire shapes, filtered fields, UTC milliseconds and replay status.
func TestReportSuccess(t *testing.T) {
	tests := []successCase{
		{name: "read pending", secure: true, want: 200},
		{name: "read previous account", account: true, secure: true, want: 200},
		{name: "new generation", post: true, secure: true, want: 201},
		{name: "original generation replay", post: true, replay: true, secure: true, want: 200},
		{name: "development cookie", want: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.h.SecureCookie = tt.secure
			f.session(t, enum.HR, nil)
			r := request(http.MethodGet, readPath, "")
			g := generation()
			if tt.post {
				r = request(http.MethodPost, postPath, "{}")
				f.allowCSRF(t, nil)
				f.regenerate.On("Execute", mock.Anything, interviewreportusecase.RegenerateInput{AssignmentID: assignmentID, IdempotencyKey: "report-1"}).Return(&interviewreportusecase.RegenerateResult{Generation: g, Replayed: tt.replay}, nil).Once()
			} else {
				view := &dto.InterviewReportView{AssignmentID: assignmentID, ExecutionID: "execution", Generation: *g}
				if tt.account {
					entry, revision := "entry", 2
					view.Report = &dto.ReportAccount{GenerationID: "previous", Observations: []dto.ReportObservation{{Body: "Described experience", EvidenceEntryID: entry, EvidenceRevision: revision}}, Gaps: []dto.ReportGap{{Body: "Not discussed"}}, Uncertainties: []dto.ReportUncertainty{{Body: "Unclear"}, {Body: "Partly clear", EvidenceEntryID: &entry, EvidenceRevision: &revision}}}
				}
				f.read.On("Execute", mock.Anything, interviewreportquery.ReadInput{AssignmentID: assignmentID}).Return(view, nil).Once()
			}
			if !tt.secure {
				r.Header.Set("Cookie", "ia-admin=session-secret")
			}
			rec := httptest.NewRecorder()
			f.h.ServeHTTP(rec, r)
			require.Equal(t, tt.want, rec.Code)
			assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Len(t, body, 2)
			assert.Equal(t, "success", body["status"])
			data := body["data"].(map[string]any)
			wireGeneration := data
			if !tt.post {
				assert.Len(t, data, 4)
				assert.Equal(t, assignmentID, data["assignmentId"])
				wireGeneration = data["generation"].(map[string]any)
				if tt.account {
					report := data["report"].(map[string]any)
					assert.Len(t, report, 4)
					assert.Equal(t, "previous", report["generationId"])
					assert.Equal(t, map[string]any{"body": "Not discussed", "questionId": nil}, report["gaps"].([]any)[0])
					assert.Equal(t, map[string]any{"body": "Unclear"}, report["uncertainties"].([]any)[0])
					assert.Equal(t, map[string]any{"body": "Described experience", "evidenceEntryId": "entry", "evidenceRevision": float64(2)}, report["observations"].([]any)[0])
				} else {
					assert.Nil(t, data["report"])
				}
			}
			assert.Equal(t, map[string]any{"id": g.ID, "number": float64(2), "state": "pending", "failureReason": nil, "createdAt": "2026-11-03T03:00:00.123Z"}, wireGeneration)
			for _, forbidden := range []string{"recommendation", "decision", "rank", "score", "salaryCeiling", "salaryCeilingRef"} {
				assert.NotContains(t, rec.Body.String(), `"`+forbidden+`"`)
			}
		})
	}
}

// errorCase defines a classified operation failure and its public transport mapping.
type errorCase struct {
	name   string
	err    error
	status int
	code   string
}

// TestReportErrors preserves error categories on both routes without exposing technical causes.
func TestReportErrors(t *testing.T) {
	tests := []errorCase{
		{name: "unbound", err: service.ErrSessionUnbound, status: 401, code: "IRP_UNAUTHENTICATED"},
		{name: "candidate", err: service.ErrCandidateSessionUnbound, status: 401, code: "IRP_UNAUTHENTICATED"},
		{name: "authentication", err: service.ErrAuthenticationFailed, status: 401, code: "IRP_UNAUTHENTICATED"},
		{name: "forbidden", err: service.ErrForbidden, status: 403, code: "IRP_FORBIDDEN"},
		{name: "invalid", err: service.ErrReportInvalidInput, status: 400, code: "INVALID_INPUT"},
		{name: "unknown assignment", err: interviewreportport.ErrNotFound, status: 404, code: "NOT_FOUND"},
		{name: "key conflict", err: service.ErrReportIdempotencyConflict, status: 409, code: "IRP_IDEMPOTENCY_CONFLICT"},
		{name: "not completed", err: service.ErrReportNotCompleted, status: 409, code: "IRP_NOT_COMPLETED"},
		{name: "no evidence", err: service.ErrReportEvidenceUnavailable, status: 409, code: "IRP_EVIDENCE_UNAVAILABLE"},
		{name: "generation active", err: interviewreport.ErrGenerationActive, status: 409, code: "IRP_GENERATION_ACTIVE"},
		{name: "store unavailable", err: interviewreportport.ErrDependencyUnavailable, status: 503, code: "IRP_DEPENDENCY_UNAVAILABLE"},
		{name: "corrupt report", err: interviewreport.ErrInvalidReport, status: 500, code: "INTERNAL_ERROR"},
		{name: "technical wrapping business error", err: errs.Technical{Cause: service.ErrForbidden}, status: 500, code: "INTERNAL_ERROR"},
		{name: "driver failure", err: errors.New("secret driver detail"), status: 500, code: "INTERNAL_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, post := range []bool{false, true} {
				f := newFixture(t)
				f.session(t, enum.HR, nil)
				r := request(http.MethodGet, readPath, "")
				if post {
					r = request(http.MethodPost, postPath, "{}")
					f.allowCSRF(t, nil)
					f.regenerate.On("Execute", mock.Anything, mock.Anything).Return(nil, tt.err).Once()
				} else {
					f.read.On("Execute", mock.Anything, mock.Anything).Return(nil, tt.err).Once()
				}
				rec := httptest.NewRecorder()
				f.h.ServeHTTP(rec, r)
				assertError(t, rec, tt.status, tt.code)
				assert.NotContains(t, rec.Body.String(), "secret driver detail")
			}
		})
	}
}

// assertError checks the standard error envelope and localization rather than a fallback code.
func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code)
	var body struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "error", body.Status)
	assert.Equal(t, map[string]any{}, body.Data)
	assert.Equal(t, code, body.Error.Code)
	assert.NotEmpty(t, body.Error.Message)
	assert.NotEqual(t, code, body.Error.Message)
	assert.False(t, strings.Contains(body.Error.Message, "secret"))
}
