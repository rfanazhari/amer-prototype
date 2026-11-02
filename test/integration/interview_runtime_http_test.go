package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

// runtimeProof owns production HTTP composition and a fake HTTP/WebSocket provider.
type runtimeProof struct {
	cfg appruntime.Config
	candidateLifecycleHTTP
	id      string
	calls   *atomic.Int32
	entered chan struct{}
	release chan struct{}
}

// runtimeAnswer decodes the successful session result without persisting or logging SDP.
type runtimeAnswer struct {
	AttemptID string `json:"attemptId"`
	SessionID string `json:"sessionId"`
	SDP       string `json:"sdp"`
	Version   int64  `json:"executionVersion"`
	Condition string `json:"sessionCondition"`
}

// newRuntimeProof pins a published HR revision before starting and mounts the actual runtime router.
func newRuntimeProof(t *testing.T, block bool) runtimeProof {
	t.Helper()
	f := newCandidateLifecycleHTTP(t)
	store := agentconfigrepository.Store{DB: f.db}
	hr := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	hrID := hr.HR.Snapshot().ID
	account := insertACPAccount(t, f.db)
	validation := insertOwnedValidation(t, store, hrID, 1, true)
	revision := publishOwned(t, store, hrID, validation, account.ID, uuid.NewString(), testdata.TokenHash)
	require.NoError(t, f.db.Exec(`UPDATE interview_assignments SET instance_id = ?, published_revision_id = ? WHERE id = ?`, hrID, revision, f.assignmentID).Error)
	calls := &atomic.Int32{}
	entered, release := make(chan struct{}, 1), make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("POST /live/sessions", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if block {
			select {
			case entered <- struct{}{}:
			default:
			}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"session": map[string]string{"id": fmt.Sprintf("provider-%d", n)}, "transport": map[string]string{"sdp": "original-answer", "type": "webrtc"}})
	})
	mux.Handle("/live/sessions/", websocket.Handler(func(c *websocket.Conn) {
		defer func() { _ = c.Close() }()
		for {
			var frame []byte
			if websocket.Message.Receive(c, &frame) != nil {
				return
			}
		}
	}))
	provider := httptest.NewServer(mux)
	t.Cleanup(provider.Close)
	cfg := appruntime.Config{AccessHTTPOrigin: "http://localhost:8080", AccessCSRFSecret: strings.Repeat("c", 32), OpenAIAPIKey: "fake-key", OpenAILiveURL: provider.URL + "/live/sessions", OpenAILiveAttachURL: "ws" + strings.TrimPrefix(provider.URL, "http") + "/live/sessions"}
	logs := logmanager.NewApplication(logmanager.WithAppName("irt-proof"), logmanager.WithSkipHeaders(), logmanager.WithMaskingConfig([]logmanager.MaskingConfig{{FieldPattern: "sdp", Type: logmanager.FullMask}, {FieldPattern: "instructions", Type: logmanager.FullMask}, {FieldPattern: "authorization", Type: logmanager.FullMask}}))
	api, _ := appruntime.NewCandidateInterview(f.db, cfg, logs)
	t.Cleanup(func() { require.NoError(t, api.(io.Closer).Close()) })
	f.api = appruntime.HTTPServer(appruntime.Router{Interview: api, Task: http.NotFoundHandler(), Access: http.NotFoundHandler()}, logs)
	consent := f.accept(t)
	started := f.request("POST", "/candidate/executions", `{"consentId":"`+consent+`"}`, "runtime-start")
	require.Equal(t, 201, started.Code, started.Body.String())
	id := data(t, started).ID
	t.Cleanup(func() {
		for _, table := range interviewTables {
			require.NoError(t, f.db.Exec(`DELETE FROM `+table+` WHERE execution_id = ?`, id).Error)
		}
		require.NoError(t, f.db.Exec(`DELETE FROM interview_progress WHERE execution_id = ?`, id).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM agent_execution_commands WHERE execution_id = ?`, id).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM agent_execution_attempts WHERE execution_id = ?`, id).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM agent_execution_operations WHERE execution_id = ?`, id).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM agent_execution_transitions WHERE execution_id = ?`, id).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM interview_controllers WHERE assignment_id = ?`, f.assignmentID).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM interview_executions WHERE id = ?`, id).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM agent_executions WHERE id = ?`, id).Error)
	})
	return runtimeProof{cfg: cfg, candidateLifecycleHTTP: f, id: id, calls: calls, entered: entered, release: release}
}

// attempt sends one fully authenticated setup request through production composition.
func (f runtimeProof) attempt(key, intent string, version int64) *httptest.ResponseRecorder {
	return f.request("POST", "/candidate/executions/"+f.id+"/session-attempts", fmt.Sprintf(`{"controllerId":"tab-proof","idempotencyKey":%q,"intent":%q,"expectedVersion":%d,"sdp":"offer"}`, key, intent, version), "")
}

// answer decodes the original successful result for identity and replay comparisons.
func answer(t *testing.T, rec *httptest.ResponseRecorder) runtimeAnswer {
	t.Helper()
	var e struct {
		Data runtimeAnswer `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
	return e.Data
}

// TestInterviewRuntimeHTTPPostgres proves production mounting, concurrent reservation and original replay.
func TestInterviewRuntimeHTTPPostgres(t *testing.T) {
	t.Run("mounted setup readiness reload and replay", func(t *testing.T) {
		f := newRuntimeProof(t, false)
		first := f.attempt("same-key", "activate", 1)
		require.Equal(t, 201, first.Code, first.Body.String())
		got := answer(t, first)
		require.Equal(t, int64(2), got.Version)
		require.Equal(t, "original-answer", got.SDP)
		require.Equal(t, int32(1), f.calls.Load())
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND provider_session_id = 'provider-1'`, f.id))
		replay := f.attempt("same-key", "activate", 1)
		require.Equal(t, 200, replay.Code, replay.Body.String())
		require.Equal(t, got, answer(t, replay))
		require.Equal(t, int32(1), f.calls.Load())
		require.Equal(t, 409, f.attempt("same-key", "reconnect", 1).Code)
		ready := f.request("POST", "/candidate/executions/"+f.id+"/session-attempts/"+got.AttemptID+"/ready", `{"controllerId":"tab-proof","expectedVersion":2}`, "")
		require.Equal(t, 200, ready.Code, ready.Body.String())
		require.Equal(t, "ready", answer(t, ready).Condition)
		unknownReady := f.request("POST", "/candidate/executions/"+f.id+"/session-attempts/unknown-attempt/ready", `{"controllerId":"tab-proof","expectedVersion":2}`, "")
		require.Equal(t, 409, unknownReady.Code, unknownReady.Body.String())
		require.Contains(t, unknownReady.Body.String(), "IRT_NOT_READY")
		require.EqualValues(t, 3, f.count(t, `SELECT count(*) FROM interview_coverage_items WHERE execution_id = ? AND state = 'pending'`, f.id))
		foreign := f.request("POST", "/candidate/executions/"+uuid.NewString()+"/session-attempts", `{"controllerId":"tab-proof","idempotencyKey":"foreign","intent":"activate","expectedVersion":1,"sdp":"offer"}`, "")
		require.Equal(t, 404, foreign.Code)
		wrong := f.request("POST", "/candidate/executions/"+f.id+"/session-attempts", `{"controllerId":"wrong-tab","idempotencyKey":"wrong","intent":"activate","expectedVersion":2,"sdp":"offer"}`, "")
		require.Equal(t, 409, wrong.Code)
		require.Equal(t, int32(1), f.calls.Load())
		extra := f.attempt("new-activation", "activate", 2)
		require.Equal(t, 409, extra.Code)
		var state string
		require.NoError(t, f.db.Raw(`SELECT state FROM agent_executions WHERE id = ?`, f.id).Scan(&state).Error)
		require.Equal(t, "RUNNING", state)
	})
	t.Run("concurrent same key has one provider create", func(t *testing.T) {
		f := newRuntimeProof(t, true)
		done := make(chan *httptest.ResponseRecorder, 1)
		go func() { done <- f.attempt("concurrent", "activate", 1) }()
		select {
		case <-f.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("provider was not reached")
		}
		pending := f.attempt("concurrent", "activate", 1)
		require.Equal(t, 409, pending.Code, pending.Body.String())
		require.Contains(t, pending.Body.String(), "IRT_NOT_READY")
		close(f.release)
		select {
		case first := <-done:
			require.Equal(t, 201, first.Code, first.Body.String())
		case <-time.After(5 * time.Second):
			t.Fatal("setup did not finish")
		}
		replay := f.attempt("concurrent", "activate", 1)
		require.Equal(t, 200, replay.Code, replay.Body.String())
		require.Equal(t, int32(1), f.calls.Load())
	})
	t.Run("result persistence failure keeps failed key and no provider result", func(t *testing.T) {
		f := newRuntimeProof(t, false)
		trigger := "irt_result_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		require.NoError(t, f.db.Exec(`CREATE FUNCTION `+trigger+`() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.state = 'succeeded' THEN RAISE EXCEPTION 'result failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER `+trigger+` BEFORE UPDATE ON interview_session_commands FOR EACH ROW EXECUTE FUNCTION `+trigger+`()`).Error)
		t.Cleanup(func() { require.NoError(t, f.db.Exec(`DROP FUNCTION `+trigger+`() CASCADE`).Error) })
		first := f.attempt("failed-result", "activate", 1)
		require.Equal(t, 500, first.Code, first.Body.String())
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_session_commands WHERE execution_id = ? AND state = 'failed' AND answer_sdp IS NULL`, f.id))
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND provider_session_id IS NOT NULL`, f.id))
		replay := f.attempt("failed-result", "activate", 1)
		require.Equal(t, 409, replay.Code)
		require.Equal(t, int32(1), f.calls.Load())
	})
}

// reloadRuntime replaces adapters while keeping only durable database state and original candidate credentials.
func (f *runtimeProof) reloadRuntime(t *testing.T) {
	t.Helper()
	logs := logmanager.NewApplication(logmanager.WithAppName("irt-reload"))
	api, _ := appruntime.NewCandidateInterview(f.db, f.cfg, logs)
	t.Cleanup(func() { require.NoError(t, api.(io.Closer).Close()) })
	f.api = appruntime.Router{Interview: api, Task: http.NotFoundHandler(), Access: http.NotFoundHandler()}
}

// TestInterviewRuntimeRestoredReservationsPostgres refuses interrupted and legacy keys after adapter reload.
func TestInterviewRuntimeRestoredReservationsPostgres(t *testing.T) {
	t.Run("restored interrupted and legacy commands never call provider", func(t *testing.T) {
		f := newRuntimeProof(t, false)
		first := f.attempt("restored", "activate", 1)
		require.Equal(t, 201, first.Code, first.Body.String())
		// Represent the durable shell left by interrupted setup; no successful replay result is available.
		require.NoError(t, f.db.Exec(`UPDATE interview_session_commands SET state = 'pending', answer_sdp = NULL WHERE execution_id = ?`, f.id).Error)
		require.NoError(t, f.db.Exec(`UPDATE interview_voice_sessions SET provider_session_id = NULL WHERE execution_id = ?`, f.id).Error)
		f.reloadRuntime(t)
		pending := f.attempt("restored", "activate", 1)
		require.Equal(t, 409, pending.Code)
		require.Contains(t, pending.Body.String(), "IRT_NOT_READY")
		require.NoError(t, f.db.Exec(`UPDATE interview_session_commands SET state = 'legacy', session_id = NULL, answer_sdp = NULL, execution_version = NULL WHERE execution_id = ?`, f.id).Error)
		legacy := f.attempt("restored", "activate", 1)
		require.Equal(t, 409, legacy.Code)
		require.Contains(t, legacy.Body.String(), "IRT_CONFLICT")
		require.Equal(t, int32(1), f.calls.Load())
	})
	t.Run("empty provider key refuses before durable attempt", func(t *testing.T) {
		f := newRuntimeProof(t, false)
		f.cfg.OpenAIAPIKey = ""
		f.reloadRuntime(t)
		refused := f.attempt("no-key", "activate", 1)
		require.Equal(t, 503, refused.Code)
		require.Contains(t, refused.Body.String(), "SESSION_UNAVAILABLE")
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_session_commands WHERE execution_id = ?`, f.id))
		require.Equal(t, int32(0), f.calls.Load())
	})
}

// TestInterviewRuntimeSingleConnectionPostgres proves admission rechecks use the already-held transaction connection.
func TestInterviewRuntimeSingleConnectionPostgres(t *testing.T) {
	f := newRuntimeProof(t, false)
	pool, err := f.db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	inner := f.api
	f.api = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		inner.ServeHTTP(w, r.WithContext(ctx))
	})
	result := f.attempt("one-connection", "activate", 1)
	require.Equal(t, 201, result.Code, result.Body.String())
}
