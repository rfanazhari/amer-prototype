package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/candidatelifecycle"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// candidateLifecycleHTTP holds the production composition over one disposable assignment and session.
type candidateLifecycleHTTP struct {
	consentFixture
	api    http.Handler
	secret string
	csrf   string
}

// candidateLifecycleData decodes only the candidate identifiers and lifecycle fields used by these proofs.
type candidateLifecycleData struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	Version int64  `json:"version"`
}

// newCandidateLifecycleHTTP connects real HTTP, authentication, consent, controller and execution adapters.
func newCandidateLifecycleHTTP(t *testing.T) candidateLifecycleHTTP {
	t.Helper()
	f := newConsentFixture(t)
	tokens := accesscredentials.Tokens{}
	secret, hash, err := tokens.Issue(context.Background(), service.CredentialCandidate)
	require.NoError(t, err)
	require.NoError(t, f.db.Exec(`UPDATE candidate_sessions SET token_hash = ? WHERE id = ?`, hash, f.sessionID).Error)
	cfg := appruntime.Config{AccessHTTPOrigin: "http://localhost:8080", AccessCSRFSecret: strings.Repeat("c", 32)}
	csrf, err := (accesscredentials.CSRF{Secret: []byte(cfg.AccessCSRFSecret)}).Issue(service.CredentialCandidate, f.sessionID)
	require.NoError(t, err)
	api, _ := appruntime.NewCandidateInterview(f.db, cfg, logmanager.NewApplication(logmanager.WithAppName("cin-proof")))
	t.Cleanup(func() {
		require.NoError(t, f.db.Exec(`DELETE FROM interview_controllers WHERE assignment_id = ?`, f.assignmentID).Error)
	})
	return candidateLifecycleHTTP{consentFixture: f, api: api, secret: secret, csrf: csrf}
}

// request executes the production candidate handler with the fixture credentials and tab.
func (f candidateLifecycleHTTP) request(method, path, body, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", f.csrf)
	req.Header.Set("X-Interview-Controller", "tab-proof")
	req.Header.Set("Idempotency-Key", key)
	req.AddCookie(&http.Cookie{Name: "ia-candidate", Value: f.secret})
	rec := httptest.NewRecorder()
	f.api.ServeHTTP(rec, req)
	return rec
}

// data decodes a successful candidate envelope without depending on private handler helpers.
func data(t *testing.T, rec *httptest.ResponseRecorder) candidateLifecycleData {
	t.Helper()
	var envelope struct {
		Data candidateLifecycleData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	return envelope.Data
}

// accept explicitly accepts the approved notice before any execution start.
func (f candidateLifecycleHTTP) accept(t *testing.T) string {
	t.Helper()
	rec := f.request(http.MethodPost, "/candidate/consents", `{"noticeVersion":"notice-261102","accepted":true}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return data(t, rec).ID
}

// TestCandidateLifecyclePostgres proves T11 through production composition and release-schema PostgreSQL.
func TestCandidateLifecyclePostgres(t *testing.T) {
	t.Run("lost response concurrency intent conflict cancel and poll", func(t *testing.T) {
		f := newCandidateLifecycleHTTP(t)
		consent := f.accept(t)
		body := `{"consentId":"` + consent + `"}`
		const callers = 8
		replies := make([]*httptest.ResponseRecorder, callers)
		var wg sync.WaitGroup
		for i := range callers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				replies[i] = f.request(http.MethodPost, "/candidate/executions", body, "start-proof")
			}()
		}
		wg.Wait()
		created := 0
		var id string
		for _, rec := range replies {
			require.Contains(t, []int{http.StatusCreated, http.StatusOK}, rec.Code, rec.Body.String())
			if rec.Code == http.StatusCreated {
				created++
			}
			got := data(t, rec)
			if id == "" {
				id = got.ID
			}
			require.Equal(t, id, got.ID)
		}
		require.Equal(t, 1, created)
		retry := f.request(http.MethodPost, "/candidate/executions", body, "start-proof")
		require.Equal(t, http.StatusOK, retry.Code, retry.Body.String())
		require.Equal(t, id, data(t, retry).ID)
		conflict := f.request(http.MethodPost, "/candidate/executions", body, "different-intent")
		require.Equal(t, http.StatusConflict, conflict.Code, conflict.Body.String())
		adapter := candidatelifecycle.Lifecycle{Assignments: f.executionFixture.store, Transactions: f.executionFixture.store, Executions: f.executionFixture.store, Commands: f.executionFixture.store, Clock: wallClock{}}
		_, err := adapter.CommitOrFind(context.Background(), service.CommitOrFindRequest{AssignmentID: f.assignmentID, ConsentID: uuid.NewString(), IdempotencyKey: "start-proof"})
		require.ErrorIs(t, err, service.ErrInterviewConflict)
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_execution_commands WHERE scope = ?`, f.assignmentID))
		stop := f.request(http.MethodPost, "/candidate/executions/"+id+"/cancel", `{"expectedVersion":1}`, "")
		require.Equal(t, http.StatusOK, stop.Code, stop.Body.String())
		require.Equal(t, "CANCELLED", data(t, stop).State)
		poll := f.request(http.MethodGet, "/candidate/executions/"+id, "", "")
		require.Equal(t, http.StatusOK, poll.Code, poll.Body.String())
		require.Equal(t, "CANCELLED", data(t, poll).State)
		require.NotContains(t, poll.Body.String(), "report")
		var reason string
		require.NoError(t, f.db.Raw(`SELECT terminal_reason FROM agent_executions WHERE id = ?`, id).Scan(&reason).Error)
		require.Equal(t, "candidate_early_stop", reason)
	})
	t.Run("coverage refused before lifecycle", func(t *testing.T) {
		f := newCandidateLifecycleHTTP(t)
		consent := f.accept(t)
		require.NoError(t, f.db.Exec(`UPDATE candidate_sessions SET expires_at = ? WHERE id = ?`, time.Now().Add(14*time.Minute), f.sessionID).Error)
		rec := f.request(http.MethodPost, "/candidate/executions", `{"consentId":"`+consent+`"}`, "coverage")
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), errs.CINSessionCoverage)
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))
	})
	t.Run("completed execution is read and never restarted", func(t *testing.T) {
		f := newCandidateLifecycleHTTP(t)
		consent := f.accept(t)
		body := `{"consentId":"` + consent + `"}`
		first := f.request(http.MethodPost, "/candidate/executions", body, "complete-proof")
		require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
		id := data(t, first).ID
		ops := newLifecycleOps(t, f.executionFixture)
		stored, err := ops.start.Execute(context.Background(), f.startInput("runtime-read"))
		require.NoError(t, err)
		progressed := ops.runToProgress(t, stored)
		adapter := candidatelifecycle.Lifecycle{Executions: f.executionFixture.store}
		running, err := adapter.Find(context.Background(), f.assignmentID)
		require.NoError(t, err)
		require.Equal(t, progressed.ActivatedAt.Add(15*time.Minute), *running.HardDeadline)
		restored, err := (candidatelifecycle.Lifecycle{Executions: f.executionFixture.store}).Find(context.Background(), f.assignmentID)
		require.NoError(t, err)
		require.Equal(t, *running.HardDeadline, *restored.HardDeadline)

		_, err = ops.complete.Execute(context.Background(), completeInput(progressed))
		require.NoError(t, err)
		again := f.request(http.MethodPost, "/candidate/executions", body, "complete-proof")
		require.Equal(t, http.StatusOK, again.Code, again.Body.String())
		require.Equal(t, id, data(t, again).ID)
		require.Equal(t, "COMPLETED", data(t, again).State)
		poll := f.request(http.MethodGet, "/candidate/executions/"+id, "", "")
		require.Equal(t, http.StatusOK, poll.Code, poll.Body.String())
		require.Equal(t, "COMPLETED", data(t, poll).State)
		require.NotContains(t, poll.Body.String(), "report")
	})
	t.Run("revocation refusal and explicit lifecycle cancellation are distinct", func(t *testing.T) {
		f := newCandidateLifecycleHTTP(t)
		consent := f.accept(t)
		first := f.request(http.MethodPost, "/candidate/executions", `{"consentId":"`+consent+`"}`, "revoke-proof")
		require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
		id := data(t, first).ID
		require.NoError(t, f.db.Exec(`UPDATE interview_assignments SET access_revoked_at = now() WHERE id = ?`, f.assignmentID).Error)
		denied := f.request(http.MethodGet, "/candidate/executions/"+id, "", "")
		require.Equal(t, http.StatusUnauthorized, denied.Code, denied.Body.String())
		before, err := f.executionFixture.store.Find(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, "CREATED", string(before.State), "401 is not a cancellation commit")
		cancel := executionusecase.CancelNonterminal{Assignments: f.executionFixture.store, Executions: f.executionFixture.store, Clock: wallClock{}}
		require.NoError(t, cancel.Execute(context.Background(), executionusecase.CancelNonterminalInput{AssignmentID: f.assignmentID, ActorRef: "access-owner", Reason: "access_revoked"}))
		after, err := f.executionFixture.store.Find(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, "CANCELLED", string(after.State))
		require.Equal(t, "access_revoked", after.TerminalReason)
	})
}
