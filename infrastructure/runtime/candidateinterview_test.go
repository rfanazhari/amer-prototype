package runtime_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// wellFormedSecret is a base64url 32-byte credential, the only shape that reaches the session lookup.
const wellFormedSecret = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// composedInterview opens the composition over sqlmock so tests own the SQL expectations.
func composedInterview(t *testing.T) (http.Handler, http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), SkipDefaultTransaction: true, DisableAutomaticPing: true,
	})
	require.NoError(t, err)
	logs := logmanager.NewApplication(logmanager.WithAppName("test"))
	api, page := appruntime.NewCandidateInterview(gdb, validAccessConfig(), logs)
	require.NotNil(t, api)
	require.NotNil(t, page)
	return api, page, mock
}

// TestNewCandidateInterviewServesPageShell verifies the composed page handler serves the shell.
func TestNewCandidateInterviewServesPageShell(t *testing.T) {
	_, page, _ := composedInterview(t)
	rec := httptest.NewRecorder()

	page.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/candidate/interview/page", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}

// TestNewCandidateInterviewRejectsUnknownSession verifies the composed read resolves the cookie through the store.
func TestNewCandidateInterviewRejectsUnknownSession(t *testing.T) {
	api, _, mock := composedInterview(t)
	mock.ExpectQuery(`SELECT .* FROM "candidate_sessions" JOIN interview_assignments .* WHERE candidate_sessions.token_hash = .* AND interview_assignments.access_revoked_at IS NULL`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	req := httptest.NewRequest(http.MethodGet, "/candidate/interview", nil)
	req.Host = "localhost:8080"
	req.AddCookie(&http.Cookie{Name: "ia-candidate", Value: wellFormedSecret})
	rec := httptest.NewRecorder()

	api.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), errs.IAUnauthenticated)
}

// TestNewCandidateInterviewOffersAcceptWithApprovedNoticeAndNoExecution verifies the production composition:
// the approved notice is loaded, and no consent or execution is stored yet.
func TestNewCandidateInterviewOffersAcceptWithApprovedNoticeAndNoExecution(t *testing.T) {
	api, _, mock := composedInterview(t)
	now := time.Now().UTC()
	mock.ExpectQuery(`SELECT .* FROM "candidate_sessions" JOIN interview_assignments .* WHERE candidate_sessions.token_hash = .* AND interview_assignments.access_revoked_at IS NULL`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "assignment_id", "invitation_id", "token_hash", "created_at", "expires_at", "revoked_at"}).
			AddRow("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeee1", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1", "hash", now, now.Add(2*time.Hour), nil))
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_consents["']?`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .* FROM "agent_executions".*JOIN interview_executions`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	req := httptest.NewRequest(http.MethodGet, "/candidate/interview", nil)
	req.Host = "localhost:8080"
	req.AddCookie(&http.Cookie{Name: "ia-candidate", Value: wellFormedSecret})
	rec := httptest.NewRecorder()

	api.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := rec.Body.String()
	require.Contains(t, body, `"version":"`+appruntime.CurrentCandidateNotice().Version+`"`)
	require.Contains(t, body, `"execution":null`)
	require.Contains(t, body, `"actions":["accept"]`)
	require.Contains(t, body, `"consent":null`)
}
