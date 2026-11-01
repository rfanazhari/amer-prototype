package runtime_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// unboundExecutionCase names a composed route that must fail closed without execution SQL writes.
type unboundExecutionCase struct {
	name    string
	path    string
	body    string
	consent bool
}

// TestCandidateExecutionCompositionFailsClosed exercises real operation wiring with SQL mocked and lifecycle unbound.
func TestCandidateExecutionCompositionFailsClosed(t *testing.T) {
	tests := []unboundExecutionCase{
		{name: "start", path: "/candidate/executions", body: `{"consentId":"11111111-1111-4111-8111-111111111111"}`, consent: true},
		{name: "cancel", path: "/candidate/executions/22222222-2222-4222-8222-222222222222/cancel", body: `{"expectedVersion":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, _, sql := composedInterview(t)
			now := time.Now().UTC().Truncate(time.Millisecond)
			sessionID := "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeee1"
			assignmentID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1"
			for range 2 {
				sql.ExpectQuery(`SELECT .* FROM "candidate_sessions"`).WillReturnRows(sqlmock.NewRows([]string{"id", "assignment_id", "invitation_id", "token_hash", "created_at", "expires_at"}).AddRow(sessionID, assignmentID, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1", "hash", now, now.Add(2*time.Hour)))
			}
			if tt.consent {
				sql.ExpectQuery(`SELECT .* FROM "interview_consents"`).WillReturnRows(sqlmock.NewRows([]string{"id", "assignment_id", "notice_version", "accepted_at", "candidate_session_id"}).AddRow("11111111-1111-4111-8111-111111111111", assignmentID, "notice-261102", now, sessionID))
			}
			cfg := validAccessConfig()
			csrf, err := (accesscredentials.CSRF{Secret: []byte(cfg.AccessCSRFSecret)}).Issue(service.CredentialCandidate, sessionID)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Origin", cfg.AccessHTTPOrigin)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", csrf)
			req.Header.Set("Idempotency-Key", "start-1")
			req.Header.Set("X-Interview-Controller", "tab-1")
			req.AddCookie(&http.Cookie{Name: "ia-candidate", Value: wellFormedSecret})
			rec := httptest.NewRecorder()
			api.ServeHTTP(rec, req)
			require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), errs.CINDependencyUnavailable)
		})
	}
}
