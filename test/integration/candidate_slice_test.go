package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// sliceView decodes the candidate-safe interview screen used by the slice proof.
type sliceView struct {
	Consent   *struct{ NoticeVersion string } `json:"consent"`
	Execution *struct {
		ID                  string `json:"id"`
		State               string `json:"state"`
		Version             int64  `json:"version"`
		IsCurrentController bool   `json:"isCurrentController"`
		ControllerVersion   int64  `json:"controllerVersion"`
	} `json:"execution"`
	Actions []string `json:"actions"`
}

// as sends one candidate request from the named browser tab with the fixture credentials.
func (f candidateLifecycleHTTP) as(tab, method, path, body, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", f.csrf)
	req.Header.Set("X-Interview-Controller", tab)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	req.AddCookie(&http.Cookie{Name: "ia-candidate", Value: f.secret})
	rec := httptest.NewRecorder()
	f.api.ServeHTTP(rec, req)
	return rec
}

// screen reads GET /candidate/interview as the named tab and decodes the allow-listed view.
func (f candidateLifecycleHTTP) screen(t *testing.T, tab string) sliceView {
	t.Helper()
	rec := f.as(tab, http.MethodGet, "/candidate/interview", "", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "report")
	require.NotContains(t, rec.Body.String(), "salary")
	var envelope struct {
		Data sliceView `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	return envelope.Data
}

// TestCandidateInterviewSlicePostgres proves the Candidate Interview capability examples (T13) end to end
// through production composition over the release schema on a disposable PostgreSQL database.
func TestCandidateInterviewSlicePostgres(t *testing.T) {
	t.Run("notice before start, lost start, refresh, takeover and early cancel", func(t *testing.T) {
		f := newCandidateLifecycleHTTP(t)
		ctx := context.Background()

		before := f.screen(t, "tab-one")
		require.Nil(t, before.Consent)
		require.Equal(t, []string{"accept"}, before.Actions)
		refused := f.as("tab-one", http.MethodPost, "/candidate/executions", `{"consentId":"00000000-0000-4000-8000-000000000000"}`, "no-consent")
		require.GreaterOrEqual(t, refused.Code, 400, refused.Body.String())
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))

		consent := f.accept(t)
		accepted := f.screen(t, "tab-one")
		require.Equal(t, "notice-261102", accepted.Consent.NoticeVersion)
		require.Equal(t, []string{"start"}, accepted.Actions)

		body := `{"consentId":"` + consent + `"}`
		first := f.as("tab-one", http.MethodPost, "/candidate/executions", body, "slice-start")
		require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
		lost := f.as("tab-one", http.MethodPost, "/candidate/executions", body, "slice-start")
		require.Equal(t, http.StatusOK, lost.Code, lost.Body.String())
		id := data(t, first).ID
		require.Equal(t, id, data(t, lost).ID)

		for range 3 {
			view := f.screen(t, "tab-one")
			require.Equal(t, id, view.Execution.ID)
			require.True(t, view.Execution.IsCurrentController)
			require.Equal(t, []string{"cancel"}, view.Actions)
		}
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))

		second := f.screen(t, "tab-two")
		require.False(t, second.Execution.IsCurrentController)
		require.Equal(t, []string{"takeover"}, second.Actions)
		notController := f.as("tab-two", http.MethodPost, "/candidate/executions/"+id+"/cancel", `{"expectedVersion":1}`, "")
		require.GreaterOrEqual(t, notController.Code, 400, notController.Body.String())
		stale := f.as("tab-two", http.MethodPost, "/candidate/executions/"+id+"/takeover", `{"controllerId":"tab-two","expectedVersion":9}`, "")
		require.Equal(t, http.StatusConflict, stale.Code, stale.Body.String())
		took := f.as("tab-two", http.MethodPost, "/candidate/executions/"+id+"/takeover", `{"controllerId":"tab-two","expectedVersion":`+strconv.FormatInt(second.Execution.ControllerVersion, 10)+`}`, "")
		require.Equal(t, http.StatusOK, took.Code, took.Body.String())
		require.False(t, f.screen(t, "tab-one").Execution.IsCurrentController)
		require.True(t, f.screen(t, "tab-two").Execution.IsCurrentController)
		replaced := f.as("tab-one", http.MethodPost, "/candidate/executions/"+id+"/cancel", `{"expectedVersion":1}`, "")
		require.GreaterOrEqual(t, replaced.Code, 400, replaced.Body.String())

		stop := f.as("tab-two", http.MethodPost, "/candidate/executions/"+id+"/cancel", `{"expectedVersion":1}`, "")
		require.Equal(t, http.StatusOK, stop.Code, stop.Body.String())
		done := f.screen(t, "tab-two")
		require.Equal(t, "CANCELLED", done.Execution.State)
		require.Empty(t, done.Actions)
		stored, err := f.executionFixture.store.Find(ctx, id)
		require.NoError(t, err)
		require.Equal(t, "candidate_early_stop", stored.TerminalReason)
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))
	})
	t.Run("coverage boundary around the fifteen minute budget", func(t *testing.T) {
		f := newCandidateLifecycleHTTP(t)
		consent := f.accept(t)
		body := `{"consentId":"` + consent + `"}`
		require.NoError(t, f.db.Exec(`UPDATE candidate_sessions SET expires_at = ? WHERE id = ?`, time.Now().Add(15*time.Minute-time.Second), f.sessionID).Error)
		require.Empty(t, f.screen(t, "tab-one").Actions)
		short := f.as("tab-one", http.MethodPost, "/candidate/executions", body, "edge")
		require.Equal(t, http.StatusForbidden, short.Code, short.Body.String())
		require.Contains(t, short.Body.String(), errs.CINSessionCoverage)
		require.NoError(t, f.db.Exec(`UPDATE candidate_sessions SET expires_at = ? WHERE id = ?`, time.Now().Add(15*time.Minute+30*time.Second), f.sessionID).Error)
		require.Equal(t, []string{"start"}, f.screen(t, "tab-one").Actions)
		ok := f.as("tab-one", http.MethodPost, "/candidate/executions", body, "edge")
		require.Equal(t, http.StatusCreated, ok.Code, ok.Body.String())
	})
	t.Run("completed interview renders while no report exists", func(t *testing.T) {
		f := newCandidateLifecycleHTTP(t)
		consent := f.accept(t)
		first := f.as("tab-one", http.MethodPost, "/candidate/executions", `{"consentId":"`+consent+`"}`, "finish")
		require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
		ops := newLifecycleOps(t, f.executionFixture)
		stored, err := ops.start.Execute(context.Background(), f.startInput("runtime-read"))
		require.NoError(t, err)
		_, err = ops.complete.Execute(context.Background(), completeInput(ops.runToProgress(t, stored)))
		require.NoError(t, err)
		view := f.screen(t, "tab-one")
		require.Equal(t, "COMPLETED", view.Execution.State)
		require.Empty(t, view.Actions)
		require.Zero(t, f.count(t, `SELECT count(*) FROM pg_tables WHERE schemaname = 'public' AND tablename LIKE '%report%'`), "no report table exists")
	})
}
