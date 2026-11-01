package integration_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentbindingusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentbindingrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newCandidateAccessHTTP installs the Interview-shaped assignment fixture and wires candidate operations.
func newCandidateAccessHTTP(t *testing.T) accessHTTP {
	t.Helper()
	graph := newAccessHTTP(t)
	installInterviewAssignmentFixture(t, graph.db)
	store := accessrepository.Store{DB: graph.db}
	hasher := accesscredentials.Passwords{}
	tokens := accesscredentials.Tokens{}
	csrf := accesscredentials.CSRF{Secret: []byte(accessSecret)}
	sessions := accessauthority.Sessions{Store: store, Tokens: tokens, Clock: graph.clock}
	candidates := accessrepository.CandidateIdentities{Store: store}
	invitations := accessrepository.InvitationSecrets{Store: store}
	candidateSecrets := accessrepository.CandidateSessionSecrets{Store: store}
	candidateSessions := accessauthority.CandidateSessions{Store: candidateSecrets, Tokens: tokens, Clock: graph.clock}
	graph.authority = accessauthority.Authority{Sessions: sessions, Clock: graph.clock}
	graph.handler = accesshandler.Handler{
		Login:           accessusecase.Login{Accounts: store, Issuance: store, Hasher: hasher, Tokens: tokens, Clock: graph.clock},
		Current:         accessquery.CurrentSession{Current: sessions, CSRF: csrf, Clock: graph.clock},
		Provision:       accessusecase.ProvisionHR{Accounts: store, Current: sessions, Hasher: hasher, Clock: graph.clock},
		Logout:          accessusecase.Logout{Current: sessions, Sessions: store, Clock: graph.clock},
		CreateCandidate: accessusecase.CreateCandidate{Candidates: candidates, Current: sessions, Clock: graph.clock},
		CreateAssignment: agentbindingusecase.CreateAssignment{
			Current: sessions, Clock: graph.clock, Candidates: candidates,
			Instances: agentconfigrepository.Store{DB: graph.db}, Assignments: agentbindingrepository.Assignments{DB: graph.db},
		},
		Candidate:        accessquery.Candidate{Candidates: candidates, Current: sessions, Clock: graph.clock},
		ListCandidates:   accessquery.ListCandidates{Candidates: candidates, Current: sessions, Clock: graph.clock},
		IssueInvitation:  accessusecase.IssueInvitation{Assignments: store, Current: sessions, Tokens: tokens, Clock: graph.clock},
		Exchange:         accessusecase.ExchangeInvitation{Invitations: invitations, Candidates: candidates, Assignments: store, Tokens: tokens, Clock: graph.clock},
		CandidateSession: accessquery.CandidateSession{Current: candidateSessions, Candidates: candidates, Assignments: store, CSRF: csrf, Clock: graph.clock},
		CSRF:             csrf,
		Sessions:         sessions,
		Origin:           accessOrigin,
		SecureCookie:     false,
		Limiter:          &accesshandler.LoginLimiter{},
		ExchangeLimiter:  &accesshandler.LoginLimiter{},
	}
	return graph
}

// installInterviewAssignmentFixture creates candidate access tables plus a disposable Interview-shaped assignment table.
func installInterviewAssignmentFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Cleanup(func() {
		require.NoError(t, db.Exec("DROP TABLE IF EXISTS candidate_sessions, interview_invitations, interview_assignments, candidates CASCADE").Error)
	})
	for _, file := range []string{"000004_candidates.up.sql", "000006_assignment_access.up.sql"} {
		if file == "000006_assignment_access.up.sql" {
			require.NoError(t, db.Exec(`CREATE TABLE interview_assignments (
    id uuid PRIMARY KEY, candidate_id uuid NOT NULL REFERENCES candidates(id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    published_revision_id uuid NOT NULL, access_revoked_at timestamptz NULL, completed_at timestamptz NULL
   )`).Error)
		}
		artifact, err := os.ReadFile("../../migrations/pgsql/" + file)
		require.NoError(t, err)
		require.NoError(t, db.Exec(string(artifact)).Error)
	}

}

// insertAssignment writes one Interview-shaped assignment row for Access collaboration.
func insertAssignment(t *testing.T, graph accessHTTP, candidateID string, revoked, completed bool) string {
	t.Helper()
	id := uuid.NewString()
	revision := uuid.NewString()
	var revokedAt, completedAt *time.Time
	now := graph.clock.Now()
	if revoked {
		revokedAt = &now
	}
	if completed {
		completedAt = &now
	}
	require.NoError(t, graph.db.Exec(
		`INSERT INTO interview_assignments (id, candidate_id, published_revision_id, access_revoked_at, completed_at) VALUES (?, ?, ?, ?, ?)`,
		id, candidateID, revision, revokedAt, completedAt,
	).Error)
	return id
}

// revokeAssignment stamps access_revoked_at on the Interview-owned assignment row.
func revokeAssignment(t *testing.T, graph accessHTTP, assignmentID string) {
	t.Helper()
	now := graph.clock.Now()
	require.NoError(t, graph.db.Exec(`UPDATE interview_assignments SET access_revoked_at = ? WHERE id = ?`, now, assignmentID).Error)
}

// countExecutions reports Interview execution tables that Access must not create.
func countExecutions(t *testing.T, graph accessHTTP) int64 {
	t.Helper()
	var n int64
	require.NoError(t, graph.db.Raw(`SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('interview_executions','consent_records','interview_runs')`).Scan(&n).Error)
	return n
}
