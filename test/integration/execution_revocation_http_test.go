package integration_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// executionRevocationHTTPCase names a nonterminal source state for the real HR command.
type executionRevocationHTTPCase struct {
	name    string
	running bool
}

// TestExecutionRevocationHTTPPostgres proves T10 through the production router, access command and lifecycle on PostgreSQL.
func TestExecutionRevocationHTTPPostgres(t *testing.T) {
	tests := []executionRevocationHTTPCase{
		{name: "created execution", running: false},
		{name: "running execution", running: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newExecutionFixture(t)
			api, cookie, csrf, actorID := executionRevocationHTTP(t, f)
			ctx := context.Background()
			ops := newLifecycleOps(t, f)
			created, err := ops.start.Execute(ctx, f.startInput("http-revocation"))
			require.NoError(t, err)
			if tt.running {
				ops.runToProgress(t, created)
			}
			before, err := f.store.Find(ctx, created.ID)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, f.db.Exec(`DELETE FROM interview_invitations WHERE assignment_id = ?`, f.assignmentID).Error)
			})

			// Both the initial issue and replacement must preserve the complete execution snapshot.
			tokens := make([]string, 0, 2)
			for range 2 {
				reply := doJSON(t, api, http.MethodPost, "/admin/interview-assignments/"+f.assignmentID+"/invitations", "{}", csrf, []*http.Cookie{cookie})
				require.Equal(t, http.StatusCreated, reply.Code)
				var invitation struct {
					Token string `json:"token"`
				}
				require.NoError(t, json.Unmarshal(decodeAccess(t, reply).Data, &invitation))
				require.NotEmpty(t, invitation.Token)
				tokens = append(tokens, invitation.Token)
				after, err := f.store.Find(ctx, created.ID)
				require.NoError(t, err)
				require.Equal(t, before, after)
			}
			require.NotEqual(t, tokens[0], tokens[1])
			require.Zero(t, f.count(t, `SELECT count(*) FROM interview_assignments WHERE id = ? AND access_revoked_at IS NOT NULL`, f.assignmentID))

			path := "/admin/interview-assignments/" + f.assignmentID + "/revocation"
			first := doJSON(t, api, http.MethodPost, path, "{}", csrf, []*http.Cookie{cookie})
			require.Equal(t, http.StatusOK, first.Code)
			firstData := decodeAccess(t, first).Data
			var marker struct {
				ID        string `json:"id"`
				RevokedAt string `json:"accessRevokedAt"`
			}
			require.NoError(t, json.Unmarshal(firstData, &marker))
			require.Equal(t, f.assignmentID, marker.ID)
			require.NotEmpty(t, marker.RevokedAt)
			cancelled, err := f.store.Find(ctx, created.ID)
			require.NoError(t, err)
			require.Equal(t, execution.StateCancelled, cancelled.State)
			require.Equal(t, "access_revoked", cancelled.TerminalReason)
			require.Equal(t, before.StateVersion+1, cancelled.StateVersion)
			require.Equal(t, before.Binding, cancelled.Binding)
			require.NotNil(t, cancelled.TerminalAt)
			var storedMarker time.Time
			require.NoError(t, f.db.Raw(`SELECT access_revoked_at FROM interview_assignments WHERE id = ?`, f.assignmentID).Scan(&storedMarker).Error)
			require.Equal(t, storedMarker.UTC().Format("2006-01-02T15:04:05.000Z"), marker.RevokedAt)
			require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_execution_transitions WHERE execution_id = ? AND to_state = 'CANCELLED' AND reason = 'access_revoked' AND actor_ref = ?`, created.ID, actorID))

			repeat := doJSON(t, api, http.MethodPost, path, "", csrf, []*http.Cookie{cookie})
			require.Equal(t, http.StatusOK, repeat.Code)
			require.JSONEq(t, string(firstData), string(decodeAccess(t, repeat).Data))
			again, err := f.store.Find(ctx, created.ID)
			require.NoError(t, err)
			require.Equal(t, cancelled, again)
			require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_execution_transitions WHERE execution_id = ? AND to_state = 'CANCELLED'`, created.ID))
			require.Zero(t, f.count(t, `SELECT count(*) FROM interview_assignments WHERE id = ? AND completed_at IS NOT NULL`, f.assignmentID))
			_, err = ops.start.Execute(ctx, f.startInput("after-http-revocation"))
			require.ErrorIs(t, err, service.ErrExecutionAssignmentUnavailable)
			require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))
		})
	}
}

// executionRevocationHTTP starts the production runtime and authenticates a real HR session through HTTP.
func executionRevocationHTTP(t *testing.T, f executionFixture) (http.Handler, *http.Cookie, string, string) {
	t.Helper()
	email := "exl-admin-" + uuid.NewString() + "@example.com"
	bootstrapAdmin(t, f.db, wallClock{}, email)
	ready := make(chan net.Addr, 1)
	result := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-result:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Error("revocation runtime did not stop")
		}
	})
	go func() {
		result <- appruntime.Run(ctx, lifecycleConfig(databaseURL(t, "exl_revocation"), "127.0.0.1:0", ready), logmanager.NewApplication(logmanager.WithAppName("exl-revocation-proof"), logmanager.WithEnvironment("production")))
	}()
	var address net.Addr
	select {
	case address = <-ready:
	case <-time.After(15 * time.Second):
		t.Fatal("revocation runtime did not become ready")
	}
	target, err := url.Parse("http://" + address.String())
	require.NoError(t, err)
	api := httputil.NewSingleHostReverseProxy(target)
	api.Transport = &http.Transport{ResponseHeaderTimeout: 5 * time.Second}
	t.Cleanup(api.Transport.(*http.Transport).CloseIdleConnections)
	adminCookie, adminCSRF := executionHTTPLogin(t, api, email)
	hrEmail := "exl-hr-" + uuid.NewString() + "@example.com"
	provision := doJSON(t, api, http.MethodPost, "/admin/hr-accounts", `{"name":"Revocation HR","email":"`+hrEmail+`","password":"`+accessPass+`"}`, adminCSRF, []*http.Cookie{adminCookie})
	require.Equal(t, http.StatusCreated, provision.Code)
	var account struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(decodeAccess(t, provision).Data, &account))
	cookie, csrf := executionHTTPLogin(t, api, hrEmail)
	return api, cookie, csrf, account.ID
}

// executionHTTPLogin obtains the real administrative cookie and session-bound CSRF value.
func executionHTTPLogin(t *testing.T, api http.Handler, email string) (*http.Cookie, string) {
	t.Helper()
	login := doJSON(t, api, http.MethodPost, "/admin/sessions", `{"email":"`+email+`","password":"`+accessPass+`"}`, "", nil)
	require.Equal(t, http.StatusCreated, login.Code)
	cookie := cookieMap(login)["ia-admin"]
	require.NotNil(t, cookie)
	session := doJSON(t, api, http.MethodGet, "/admin/session", "", "", []*http.Cookie{cookie})
	require.Equal(t, http.StatusOK, session.Code)
	var view struct {
		CSRF string `json:"csrfToken"`
	}
	require.NoError(t, json.Unmarshal(decodeAccess(t, session).Data, &view))
	require.NotEmpty(t, view.CSRF)
	return cookie, view.CSRF
}
